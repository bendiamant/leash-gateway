import { NextResponse } from 'next/server';
import { Pool } from 'pg';

export const runtime = 'nodejs'; // Need Node.js runtime for pg

// PostgreSQL connection
const pool = new Pool({
  host: process.env.PG_HOST || 'localhost',
  port: parseInt(process.env.PG_PORT || '5433'),
  database: process.env.PG_DATABASE || 'leash',
  user: process.env.PG_USER || 'leash',
  password: process.env.PG_PASSWORD || 'leash',
});

export async function GET(req: Request) {
  const { searchParams } = new URL(req.url);
  const timeRange = searchParams.get('range') || '1h';
  
  try {
    // Get request counts and metrics per provider
    const metricsQuery = `
      SELECT 
        provider,
        COUNT(*) as request_count,
        AVG(processing_time_ms) as avg_latency,
        MAX(processing_time_ms) as max_latency,
        MIN(processing_time_ms) as min_latency,
        COUNT(CASE WHEN status_code >= 400 THEN 1 END) as error_count
      FROM audit_logs
      WHERE created_at > NOW() - INTERVAL '${timeRange}'
      GROUP BY provider
      ORDER BY request_count DESC
    `;
    
    const recentQuery = `
      SELECT 
        request_id,
        provider,
        method,
        path,
        status_code,
        processing_time_ms,
        created_at
      FROM audit_logs
      ORDER BY created_at DESC
      LIMIT 10
    `;
    
    const [metrics, recent] = await Promise.all([
      pool.query(metricsQuery),
      pool.query(recentQuery)
    ]);
    
    return NextResponse.json({
      success: true,
      data: {
        summary: metrics.rows,
        recent: recent.rows,
        timestamp: new Date().toISOString()
      }
    });
  } catch (error) {
    console.error('PostgreSQL metrics error:', error);
    
    // Return empty data on error
    return NextResponse.json({
      success: false,
      error: error instanceof Error ? error.message : 'Database connection failed',
      data: {
        summary: [],
        recent: []
      }
    });
  }
}

// Also support POST to write metrics
export async function POST(req: Request) {
  try {
    const body = await req.json();
    const {
      request_id,
      tenant_id = 'default',
      provider,
      method,
      path,
      status_code,
      processing_time_ms
    } = body;
    
    const insertQuery = `
      INSERT INTO audit_logs 
      (request_id, tenant_id, provider, method, path, status_code, processing_time_ms)
      VALUES ($1, $2, $3, $4, $5, $6, $7)
      RETURNING *
    `;
    
    const result = await pool.query(insertQuery, [
      request_id,
      tenant_id,
      provider,
      method,
      path,
      status_code,
      processing_time_ms
    ]);
    
    return NextResponse.json({
      success: true,
      data: result.rows[0]
    });
  } catch (error) {
    console.error('Failed to insert metric:', error);
    return NextResponse.json({
      success: false,
      error: error instanceof Error ? error.message : 'Insert failed'
    }, { status: 500 });
  }
}

