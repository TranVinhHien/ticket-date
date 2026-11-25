import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { authGuard } from './auth.guard';
import { AuthService } from '../services/auth.service';

describe('authGuard', () => {
  let authServiceSpy: { isAuthenticated: () => boolean };
  let routerSpy: { navigate: (commands: any[], extras?: any) => Promise<boolean> };

  beforeEach(() => {
    authServiceSpy = {
      isAuthenticated: () => false
    };
    routerSpy = {
      navigate: () => Promise.resolve(true)
    };

    TestBed.configureTestingModule({
      providers: [
        { provide: AuthService, useValue: authServiceSpy },
        { provide: Router, useValue: routerSpy }
      ]
    });
  });

  it('should allow activation when user is authenticated', () => {
    authServiceSpy.isAuthenticated = () => true;

    const result = TestBed.runInInjectionContext(() =>
      authGuard({} as any, { url: '/checkout' } as any)
    );

    expect(result).toBe(true);
  });

  it('should block activation and redirect to /login with returnUrl when user is not authenticated', () => {
    authServiceSpy.isAuthenticated = () => false;
    let navigatedTo = '';
    let navParams: any = null;
    routerSpy.navigate = (commands: any[], extras: any) => {
      navigatedTo = commands[0];
      navParams = extras;
      return Promise.resolve(true);
    };

    const result = TestBed.runInInjectionContext(() =>
      authGuard({} as any, { url: '/checkout' } as any)
    );

    expect(result).toBe(false);
    expect(navigatedTo).toBe('/login');
    expect(navParams?.queryParams?.returnUrl).toBe('/checkout');
  });
});
