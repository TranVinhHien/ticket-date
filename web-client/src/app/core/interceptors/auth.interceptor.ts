import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, switchMap, throwError } from 'rxjs';
import { AuthService } from '../services/auth.service';
import { AuthApiService } from '../../services/auth-api.service';

let isRefreshing = false;

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const authApi = inject(AuthApiService);
  const token = authService.token();

  let authReq = req;
  if (token && !req.headers.has('Authorization')) {
    authReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // Handle 401 Unauthorized
      if (error.status === 401 && !req.url.includes('/auth/')) {
        const refreshToken = authService.refreshToken();
        if (refreshToken && !isRefreshing) {
          isRefreshing = true;
          return authApi.refreshToken(refreshToken).pipe(
            switchMap(res => {
              isRefreshing = false;
              if (res.data) {
                authService.saveSession(res.data);
                const retryReq = req.clone({
                  setHeaders: {
                    Authorization: `Bearer ${res.data.token}`
                  }
                });
                return next(retryReq);
              }
              authService.logout();
              return throwError(() => error);
            }),
            catchError(refreshErr => {
              isRefreshing = false;
              authService.logout();
              return throwError(() => refreshErr);
            })
          );
        } else {
          authService.logout();
        }
      }
      return throwError(() => error);
    })
  );
};
