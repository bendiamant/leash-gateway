import { LeashError } from '../types';

export class LeashErrorHandler {
  static handleError(error: any, context: {
    requestId?: string;
    provider?: string;
    model?: string;
  }): LeashError {
    const leashError: LeashError = new Error() as LeashError;
    
    // Handle Axios errors
    if (error.response) {
      // HTTP error response
      leashError.message = `HTTP ${error.response.status}: ${error.response.statusText}`;
      leashError.code = `HTTP_${error.response.status}`;
      leashError.status = error.response.status;
      leashError.details = {
        response: error.response.data,
        headers: error.response.headers,
      };

      // Specific error handling based on status code
      switch (error.response.status) {
        case 401:
          leashError.message = 'Authentication failed. Please check your API key.';
          leashError.code = 'AUTHENTICATION_ERROR';
          break;
        case 403:
          leashError.message = 'Request blocked by security policy.';
          leashError.code = 'POLICY_VIOLATION';
          break;
        case 429:
          leashError.message = 'Rate limit exceeded. Please try again later.';
          leashError.code = 'RATE_LIMIT_EXCEEDED';
          break;
        case 500:
        case 502:
        case 503:
        case 504:
          leashError.message = 'Provider service unavailable. Please try again.';
          leashError.code = 'PROVIDER_UNAVAILABLE';
          break;
      }
    } else if (error.request) {
      // Network error
      leashError.message = 'Network error: Unable to reach the gateway.';
      leashError.code = 'NETWORK_ERROR';
      leashError.details = { request: error.request };
    } else {
      // Other error
      leashError.message = error.message || 'Unknown error occurred';
      leashError.code = 'UNKNOWN_ERROR';
    }

    // Add context
    leashError.provider = context.provider;
    leashError.requestId = context.requestId;
    if (context.model) {
      leashError.details = { ...leashError.details, model: context.model };
    }

    return leashError;
  }

  static isRetryableError(error: LeashError): boolean {
    const retryableCodes = [
      'NETWORK_ERROR',
      'PROVIDER_UNAVAILABLE',
      'HTTP_500',
      'HTTP_502',
      'HTTP_503',
      'HTTP_504',
    ];

    return retryableCodes.includes(error.code);
  }

  static isClientError(error: LeashError): boolean {
    const clientErrorCodes = [
      'AUTHENTICATION_ERROR',
      'POLICY_VIOLATION',
      'HTTP_400',
      'HTTP_401',
      'HTTP_403',
      'HTTP_404',
    ];

    return clientErrorCodes.includes(error.code);
  }

  static shouldFallback(error: LeashError): boolean {
    // Don't fallback on client errors or policy violations
    if (this.isClientError(error)) {
      return false;
    }

    // Fallback on provider issues
    const fallbackCodes = [
      'PROVIDER_UNAVAILABLE',
      'NETWORK_ERROR',
      'HTTP_500',
      'HTTP_502',
      'HTTP_503',
      'HTTP_504',
    ];

    return fallbackCodes.includes(error.code);
  }
}

// Predefined error types
export class AuthenticationError extends Error implements LeashError {
  code = 'AUTHENTICATION_ERROR';
  status = 401;
  provider?: string;
  requestId?: string;
  details?: Record<string, any>;

  constructor(message: string = 'Authentication failed') {
    super(message);
    this.name = 'AuthenticationError';
  }
}

export class PolicyViolationError extends Error implements LeashError {
  code = 'POLICY_VIOLATION';
  status = 403;
  provider?: string;
  requestId?: string;
  details?: Record<string, any>;

  constructor(message: string = 'Request blocked by security policy') {
    super(message);
    this.name = 'PolicyViolationError';
  }
}

export class RateLimitError extends Error implements LeashError {
  code = 'RATE_LIMIT_EXCEEDED';
  status = 429;
  provider?: string;
  requestId?: string;
  details?: Record<string, any>;

  constructor(message: string = 'Rate limit exceeded') {
    super(message);
    this.name = 'RateLimitError';
  }
}

export class ProviderUnavailableError extends Error implements LeashError {
  code = 'PROVIDER_UNAVAILABLE';
  status = 503;
  provider?: string;
  requestId?: string;
  details?: Record<string, any>;

  constructor(message: string = 'Provider service unavailable') {
    super(message);
    this.name = 'ProviderUnavailableError';
  }
}

export class NetworkError extends Error implements LeashError {
  code = 'NETWORK_ERROR';
  provider?: string;
  requestId?: string;
  details?: Record<string, any>;

  constructor(message: string = 'Network error occurred') {
    super(message);
    this.name = 'NetworkError';
  }
}
