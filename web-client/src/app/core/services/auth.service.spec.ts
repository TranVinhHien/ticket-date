import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { AuthService } from './auth.service';
import { StorageService } from './storage.service';
import { AuthResponse } from '../models/auth.model';

describe('AuthService', () => {
  let service: AuthService;
  let storage: StorageService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        AuthService,
        StorageService
      ]
    });
    service = TestBed.inject(AuthService);
    storage = TestBed.inject(StorageService);
    storage.removeItem('td_access_token');
    storage.removeItem('td_refresh_token');
    storage.removeItem('td_user');
  });

  it('should be initialized unauthenticated when storage is empty', () => {
    expect(service.isAuthenticated()).toBe(false);
    expect(service.token()).toBeNull();
    expect(service.currentUser()).toBeNull();
  });

  it('should save session and update authentication status', () => {
    const mockAuth: AuthResponse = {
      token: 'jwt-access-token',
      refresh_token: 'jwt-refresh-token',
      session_id: 'session-123',
      user: {
        id: 'user-123',
        email: 'test@example.com',
        roles: ['user'],
        provider: 'local',
        is_active: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    };

    service.saveSession(mockAuth);

    expect(service.isAuthenticated()).toBe(true);
    expect(service.token()).toBe('jwt-access-token');
    expect(service.currentUser()?.email).toBe('test@example.com');
  });

  it('should clear session on logout', () => {
    const mockAuth: AuthResponse = {
      token: 'jwt-access-token',
      refresh_token: 'jwt-refresh-token',
      session_id: 'session-123',
      user: {
        id: 'user-123',
        email: 'test@example.com',
        roles: ['user'],
        provider: 'local',
        is_active: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
    };

    service.saveSession(mockAuth);
    expect(service.isAuthenticated()).toBe(true);

    service.clearSession();
    expect(service.isAuthenticated()).toBe(false);
    expect(service.token()).toBeNull();
    expect(service.currentUser()).toBeNull();
  });
});
