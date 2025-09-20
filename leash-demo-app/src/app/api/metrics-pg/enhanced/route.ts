import { NextResponse } from 'next/server';
import { Pool } from 'pg';

export const runtime = 'nodejs';

const pool = new Pool({
  host: process.env.PG_HOST || 'localhost',
  port: parseInt(process.env.PG_PORT || '5433'),
  database: process.env.PG_DATABASE || 'leash',
  user: process.env.PG_USER || 'leash',
  password: process.env.PG_PASSWORD || 'leash',
});

// Handle enhanced metrics write
export async function POST(req: Request) {
  const client = await pool.connect();
  
  try {
    const body = await req.json();
    const { audit, request_body, response_body, messages } = body;
    
    // Start a transaction for atomic writes
    await client.query('BEGIN');
    
    // 1. Insert into audit_logs with enhanced data
    const auditResult = await client.query(`
      INSERT INTO audit_logs (
        request_id, tenant_id, provider, model, method, path,
        status_code, processing_time_ms,
        prompt_tokens, completion_tokens, total_tokens,
        cost_usd, temperature, max_tokens
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
      RETURNING id
    `, [
      audit.request_id,
      audit.tenant_id || 'default',
      audit.provider,
      audit.model,
      audit.method,
      audit.path,
      audit.status_code,
      audit.processing_time_ms,
      audit.prompt_tokens,
      audit.completion_tokens,
      audit.total_tokens,
      audit.cost_usd,
      audit.temperature,
      audit.max_tokens
    ]);
    
    const auditLogId = auditResult.rows[0].id;
    
    // 2. Insert request/response logs
    await client.query(`
      INSERT INTO request_logs (
        audit_log_id, request_body, response_body
      ) VALUES ($1, $2, $3)
    `, [
      auditLogId,
      JSON.stringify(request_body),
      JSON.stringify(response_body)
    ]);
    
    // 3. Insert message logs for analytics
    if (messages && Array.isArray(messages)) {
      for (let i = 0; i < messages.length; i++) {
        const msg = messages[i];
        await client.query(`
          INSERT INTO message_logs (
            audit_log_id, message_index, role, content
          ) VALUES ($1, $2, $3, $4)
        `, [
          auditLogId,
          i,
          msg.role,
          msg.content || msg.text || ''
        ]);
      }
    }
    
    // 4. Update cost tracking
    await client.query(`
      INSERT INTO cost_tracking (
        provider, model, date,
        total_requests, total_prompt_tokens, total_completion_tokens, total_cost_usd
      ) VALUES ($1, $2, CURRENT_DATE, 1, $3, $4, $5)
      ON CONFLICT (provider, model, date) DO UPDATE SET
        total_requests = cost_tracking.total_requests + 1,
        total_prompt_tokens = cost_tracking.total_prompt_tokens + $3,
        total_completion_tokens = cost_tracking.total_completion_tokens + $4,
        total_cost_usd = cost_tracking.total_cost_usd + $5,
        updated_at = NOW()
    `, [
      audit.provider,
      audit.model,
      audit.prompt_tokens || 0,
      audit.completion_tokens || 0,
      audit.cost_usd || 0
    ]);
    
    // Commit the transaction
    await client.query('COMMIT');
    
    console.log(`✅ Enhanced metrics stored: ${audit.provider}/${audit.model} - ${audit.total_tokens} tokens, $${audit.cost_usd?.toFixed(6) || '0'}`);
    
    return NextResponse.json({
      success: true,
      audit_log_id: auditLogId,
      summary: {
        provider: audit.provider,
        model: audit.model,
        tokens: audit.total_tokens,
        cost: audit.cost_usd,
        latency: audit.processing_time_ms
      }
    });
    
  } catch (error) {
    await client.query('ROLLBACK');
    console.error('Failed to write enhanced metrics:', error);
    
    return NextResponse.json({
      success: false,
      error: error instanceof Error ? error.message : 'Database error'
    }, { status: 500 });
  } finally {
    client.release();
  }
}

// Get enhanced metrics with full details
export async function GET(req: Request) {
  const { searchParams } = new URL(req.url);
  const timeRange = searchParams.get('range') || '1h';
  const includeMessages = searchParams.get('messages') === 'true';
  
  try {
    // Get summary with cost data
    const summaryQuery = `
      SELECT 
        al.provider,
        al.model,
        COUNT(*) as request_count,
        AVG(al.processing_time_ms) as avg_latency,
        SUM(al.total_tokens) as total_tokens,
        SUM(al.cost_usd) as total_cost,
        AVG(al.prompt_tokens) as avg_prompt_tokens,
        AVG(al.completion_tokens) as avg_completion_tokens
      FROM audit_logs al
      WHERE al.created_at > NOW() - INTERVAL '${timeRange}'
        AND al.model IS NOT NULL
      GROUP BY al.provider, al.model
      ORDER BY total_cost DESC NULLS LAST
    `;
    
    // Get recent requests with full details
    const recentQuery = `
      SELECT 
        al.*,
        rl.request_body,
        rl.response_body
      FROM audit_logs al
      LEFT JOIN request_logs rl ON rl.audit_log_id = al.id
      WHERE al.model IS NOT NULL
      ORDER BY al.created_at DESC
      LIMIT 10
    `;
    
    // Get cost breakdown by day
    const costQuery = `
      SELECT 
        date,
        provider,
        model,
        total_requests,
        total_prompt_tokens,
        total_completion_tokens,
        total_cost_usd
      FROM cost_tracking
      WHERE date >= CURRENT_DATE - INTERVAL '7 days'
      ORDER BY date DESC, total_cost_usd DESC
    `;
    
    const [summary, recent, costs] = await Promise.all([
      pool.query(summaryQuery),
      pool.query(recentQuery),
      pool.query(costQuery)
    ]);
    
    // Optionally get message history
    let messageHistory = [];
    if (includeMessages && recent.rows.length > 0) {
      const auditIds = recent.rows.map(r => r.id);
      const messagesResult = await pool.query(`
        SELECT * FROM message_logs
        WHERE audit_log_id = ANY($1)
        ORDER BY audit_log_id, message_index
      `, [auditIds]);
      messageHistory = messagesResult.rows;
    }
    
    return NextResponse.json({
      success: true,
      data: {
        summary: summary.rows,
        recent: recent.rows,
        costs: costs.rows,
        messages: messageHistory,
        timestamp: new Date().toISOString()
      }
    });
    
  } catch (error) {
    console.error('Failed to fetch enhanced metrics:', error);
    return NextResponse.json({
      success: false,
      error: error instanceof Error ? error.message : 'Database error',
      data: {
        summary: [],
        recent: [],
        costs: []
      }
    });
  }
}

