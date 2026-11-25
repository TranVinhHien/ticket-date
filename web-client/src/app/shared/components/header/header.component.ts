import { Component, inject } from '@angular/core';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../../core/services/auth.service';

@Component({
  selector: 'app-header',
  standalone: true,
  imports: [RouterLink, RouterLinkActive],
  template: `
    <header class="sticky top-0 z-50 bg-white/95 backdrop-blur-md border-b border-[#E1E5EE]">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
          <!-- Logo & Brand -->
          <div class="flex items-center gap-8">
            <a routerLink="/" class="flex items-center gap-2.5 group">
              <div class="w-9 h-9 rounded-lg bg-[#121622] flex items-center justify-center text-white group-hover:bg-[#1B4BFF] transition-colors">
                <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
                </svg>
              </div>
              <span class="font-display text-xl font-bold tracking-tight text-[#121622]">TicketDate</span>
            </a>

            <!-- Desktop Nav Links -->
            <nav class="hidden md:flex items-center gap-1">
              <a
                routerLink="/"
                routerLinkActive="text-[#1B4BFF] bg-[#1B4BFF]/8 font-semibold"
                [routerLinkActiveOptions]="{ exact: true }"
                class="px-3.5 py-2 rounded-lg text-sm font-medium text-[#5A6275] hover:text-[#121622] hover:bg-[#F5F6F8] transition-colors">
                Trang chủ
              </a>
              @if (authService.isAuthenticated()) {
                <a
                  routerLink="/my-tickets"
                  routerLinkActive="text-[#1B4BFF] bg-[#1B4BFF]/8 font-semibold"
                  class="px-3.5 py-2 rounded-lg text-sm font-medium text-[#5A6275] hover:text-[#121622] hover:bg-[#F5F6F8] transition-colors">
                  Vé của tôi
                </a>
              }
            </nav>
          </div>

          <!-- Auth Status & Actions -->
          <div class="flex items-center gap-3">
            @if (authService.isAuthenticated()) {
              <div class="flex items-center gap-2.5">
                <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[#F5F6F8] border border-[#E1E5EE]">
                  <div class="w-6 h-6 rounded-md bg-[#121622] text-white flex items-center justify-center text-xs font-bold uppercase">
                    {{ (authService.currentUser()?.profile?.full_name || authService.currentUser()?.email || 'U')[0] }}
                  </div>
                  <span class="text-sm font-medium text-[#121622] max-w-[130px] truncate">
                    {{ authService.currentUser()?.profile?.full_name || authService.currentUser()?.email }}
                  </span>
                </div>
                <button
                  (click)="logout()"
                  class="p-2 text-[#5A6275] hover:text-[#E11D48] hover:bg-[#FFF1F2] rounded-lg border border-transparent hover:border-[#FECDD3] transition-colors"
                  title="Đăng xuất">
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                </button>
              </div>
            } @else {
              <a
                routerLink="/login"
                class="inline-flex items-center justify-center px-4 py-2 text-sm font-semibold rounded-lg text-white bg-[#1B4BFF] hover:bg-[#143AD6] transition-colors">
                Đăng nhập
              </a>
            }
          </div>
        </div>
      </div>
    </header>
  `
})
export class HeaderComponent {
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  logout(): void {
    this.authService.logout();
  }
}
