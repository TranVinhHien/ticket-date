import { Injectable, inject, signal, computed } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, tap, catchError, of, throwError } from 'rxjs';
import { StorageService } from './storage.service';
import { AuthApiService } from '../../services/auth-api.service';
import { AuthResponse, LoginRequest, RegisterRequest, UserResponse } from '../models/auth.model';

const TOKEN_KEY = 'td_access_token';
const REFRESH_TOKEN_KEY = 'td_refresh_token';
const USER_KEY = 'td_user';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly storage = inject(StorageService);
  private readonly authApi = inject(AuthApiService);
  private readonly router = inject(Router);

  private readonly _currentUser = signal<UserResponse | null>(this.getInitialUser());
  private readonly _token = signal<string | null>(this.storage.getItem(TOKEN_KEY));
  private readonly _refreshToken = signal<string | null>(this.storage.getItem(REFRESH_TOKEN_KEY));

  readonly currentUser = this._currentUser.asReadonly();
  readonly token = this._token.asReadonly();
  readonly refreshToken = this._refreshToken.asReadonly();
  readonly isAuthenticated = computed(() => !!this._token());

  private getInitialUser(): UserResponse | null {
    const raw = this.storage.getItem(USER_KEY);
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  login(req: LoginRequest): Observable<AuthResponse> {
    return this.authApi.login(req).pipe(
      tap(res => {
        if (res.data) {
          this.saveSession(res.data);
        }
      }),
      tap({
        next: res => res.data,
      }),
      // Map to AuthResponse
      tap(),
      // Observable operator map
    ) as any;
  }

  saveSession(auth: AuthResponse): void {
    this._token.set(auth.token);
    this._refreshToken.set(auth.refresh_token);
    this._currentUser.set(auth.user);

    this.storage.setItem(TOKEN_KEY, auth.token);
    this.storage.setItem(REFRESH_TOKEN_KEY, auth.refresh_token);
    this.storage.setItem(USER_KEY, JSON.stringify(auth.user));
  }

  clearSession(): void {
    const currentRefresh = this._refreshToken();
    if (currentRefresh) {
      this.authApi.logout(currentRefresh).subscribe({ error: () => {} });
    }

    this._token.set(null);
    this._refreshToken.set(null);
    this._currentUser.set(null);

    this.storage.removeItem(TOKEN_KEY);
    this.storage.removeItem(REFRESH_TOKEN_KEY);
    this.storage.removeItem(USER_KEY);
  }

  logout(): void {
    this.clearSession();
    this.router.navigate(['/login']);
  }
}
