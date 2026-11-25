import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { AuthApiService } from '../../services/auth-api.service';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="min-h-[80vh] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div class="max-w-md w-full bg-white rounded-2xl p-7 sm:p-8 border border-[#E1E5EE]">
        <!-- Logo Header -->
        <div class="text-center mb-7">
          <div class="w-11 h-11 rounded-xl bg-[#121622] text-white flex items-center justify-center mx-auto mb-3">
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
            </svg>
          </div>
          <h2 class="font-display text-2xl font-bold text-[#121622]">
            {{ activeTab() === 'login' ? 'Đăng nhập vào Ticket Date' : 'Tạo tài khoản mới' }}
          </h2>
          <p class="text-sm text-[#5A6275] mt-1">
            {{ activeTab() === 'login' ? 'Đăng nhập để đặt vé và theo dõi đơn hàng của bạn' : 'Điền thông tin để bắt đầu đặt vé sự kiện' }}
          </p>
        </div>

        <!-- Mode Toggle Tabs -->
        <div class="flex p-1 bg-[#F5F6F8] border border-[#E1E5EE] rounded-xl mb-6">
          <button
            type="button"
            (click)="setTab('login')"
            [class]="activeTab() === 'login' ? 'bg-white text-[#121622] border border-[#E1E5EE] font-semibold' : 'text-[#5A6275] hover:text-[#121622] font-medium border border-transparent'"
            class="flex-1 py-2 text-sm rounded-lg transition-colors text-center">
            Đăng nhập
          </button>
          <button
            type="button"
            (click)="setTab('register')"
            [class]="activeTab() === 'register' ? 'bg-white text-[#121622] border border-[#E1E5EE] font-semibold' : 'text-[#5A6275] hover:text-[#121622] font-medium border border-transparent'"
            class="flex-1 py-2 text-sm rounded-lg transition-colors text-center">
            Đăng ký
          </button>
        </div>

        <!-- Error Alert -->
        @if (errorMessage()) {
          <div class="mb-5 p-3.5 bg-[#FFF1F2] border border-[#FECDD3] rounded-xl text-[#9F1239] text-sm flex items-center gap-2">
            <svg class="w-5 h-5 shrink-0 text-[#E11D48]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{{ errorMessage() }}</span>
          </div>
        }

        <!-- Success Alert for Register -->
        @if (successMessage()) {
          <div class="mb-5 p-3.5 bg-[#F0FDFA] border border-[#99F6E4] rounded-xl text-[#115E59] text-sm flex items-center gap-2">
            <svg class="w-5 h-5 shrink-0 text-[#0D9488]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
            <span>{{ successMessage() }}</span>
          </div>
        }

        <!-- Forms -->
        @if (activeTab() === 'login') {
          <form (ngSubmit)="onLogin()" class="space-y-4">
            <div>
              <label class="block text-xs font-semibold text-[#121622] mb-1.5">Email</label>
              <input
                type="email"
                [(ngModel)]="loginEmail"
                name="email"
                required
                placeholder="name@example.com"
                class="w-full px-3.5 py-2.5 rounded-xl border border-[#E1E5EE] bg-white text-[#121622] placeholder-[#7C8498] focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] focus:border-transparent text-sm transition-colors" />
            </div>

            <div>
              <label class="block text-xs font-semibold text-[#121622] mb-1.5">Mật khẩu</label>
              <input
                type="password"
                [(ngModel)]="loginPassword"
                name="password"
                required
                placeholder="••••••••"
                class="w-full px-3.5 py-2.5 rounded-xl border border-[#E1E5EE] bg-white text-[#121622] placeholder-[#7C8498] focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] focus:border-transparent text-sm transition-colors" />
            </div>

            <button
              type="submit"
              [disabled]="loading()"
              class="w-full mt-2 py-3 px-4 rounded-xl text-sm font-semibold text-white bg-[#1B4BFF] hover:bg-[#143AD6] disabled:opacity-60 transition-colors flex items-center justify-center gap-2">
              @if (loading()) {
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Đang xử lý...</span>
              } @else {
                <span>Đăng nhập</span>
              }
            </button>
          </form>
        } @else {
          <form (ngSubmit)="onRegister()" class="space-y-4">
            <div>
              <label class="block text-xs font-semibold text-[#121622] mb-1.5">Họ và tên</label>
              <input
                type="text"
                [(ngModel)]="registerFullName"
                name="fullName"
                required
                placeholder="Nguyễn Văn A"
                class="w-full px-3.5 py-2.5 rounded-xl border border-[#E1E5EE] bg-white text-[#121622] placeholder-[#7C8498] focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] focus:border-transparent text-sm transition-colors" />
            </div>

            <div>
              <label class="block text-xs font-semibold text-[#121622] mb-1.5">Email</label>
              <input
                type="email"
                [(ngModel)]="registerEmail"
                name="email"
                required
                placeholder="name@example.com"
                class="w-full px-3.5 py-2.5 rounded-xl border border-[#E1E5EE] bg-white text-[#121622] placeholder-[#7C8498] focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] focus:border-transparent text-sm transition-colors" />
            </div>

            <div>
              <label class="block text-xs font-semibold text-[#121622] mb-1.5">Mật khẩu (ít nhất 6 ký tự)</label>
              <input
                type="password"
                [(ngModel)]="registerPassword"
                name="password"
                required
                minlength="6"
                placeholder="••••••••"
                class="w-full px-3.5 py-2.5 rounded-xl border border-[#E1E5EE] bg-white text-[#121622] placeholder-[#7C8498] focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] focus:border-transparent text-sm transition-colors" />
            </div>

            <div>
              <label class="block text-xs font-semibold text-[#121622] mb-1.5">Số điện thoại (tùy chọn)</label>
              <input
                type="tel"
                [(ngModel)]="registerPhone"
                name="phone"
                placeholder="0912345678"
                class="w-full px-3.5 py-2.5 rounded-xl border border-[#E1E5EE] bg-white text-[#121622] placeholder-[#7C8498] focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] focus:border-transparent text-sm transition-colors" />
            </div>

            <button
              type="submit"
              [disabled]="loading()"
              class="w-full mt-2 py-3 px-4 rounded-xl text-sm font-semibold text-white bg-[#1B4BFF] hover:bg-[#143AD6] disabled:opacity-60 transition-colors flex items-center justify-center gap-2">
              @if (loading()) {
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Đang xử lý...</span>
              } @else {
                <span>Đăng ký tài khoản</span>
              }
            </button>
          </form>
        }
      </div>
    </div>
  `
})
export class LoginPageComponent {
  private readonly authService = inject(AuthService);
  private readonly authApi = inject(AuthApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly activeTab = signal<'login' | 'register'>('login');
  readonly loading = signal<boolean>(false);
  readonly errorMessage = signal<string>('');
  readonly successMessage = signal<string>('');

  // Login fields
  loginEmail = '';
  loginPassword = '';

  // Register fields
  registerFullName = '';
  registerEmail = '';
  registerPassword = '';
  registerPhone = '';

  setTab(tab: 'login' | 'register'): void {
    this.activeTab.set(tab);
    this.errorMessage.set('');
    this.successMessage.set('');
  }

  onLogin(): void {
    if (!this.loginEmail || !this.loginPassword) {
      this.errorMessage.set('Vui lòng nhập đầy đủ email và mật khẩu.');
      return;
    }

    this.loading.set(true);
    this.errorMessage.set('');

    this.authApi.login({ email: this.loginEmail, password: this.loginPassword }).subscribe({
      next: (res) => {
        this.loading.set(false);
        if (res.data) {
          this.authService.saveSession(res.data);
          const returnUrl = this.route.snapshot.queryParams['returnUrl'] || '/';
          this.router.navigateByUrl(returnUrl);
        }
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMessage.set(err.message || 'Email hoặc mật khẩu không chính xác.');
      }
    });
  }

  onRegister(): void {
    if (!this.registerFullName || !this.registerEmail || !this.registerPassword) {
      this.errorMessage.set('Vui lòng điền đầy đủ các thông tin bắt buộc.');
      return;
    }
    if (this.registerPassword.length < 6) {
      this.errorMessage.set('Mật khẩu phải có ít nhất 6 ký tự.');
      return;
    }

    this.loading.set(true);
    this.errorMessage.set('');

    this.authApi.register({
      full_name: this.registerFullName,
      email: this.registerEmail,
      password: this.registerPassword,
      phone_number: this.registerPhone || undefined
    }).subscribe({
      next: () => {
        this.loading.set(false);
        this.successMessage.set('Đăng ký tài khoản thành công! Bạn có thể đăng nhập ngay bây giờ.');
        this.loginEmail = this.registerEmail;
        this.loginPassword = this.registerPassword;
        this.activeTab.set('login');
      },
      error: (err) => {
        this.loading.set(false);
        this.errorMessage.set(err.message || 'Đăng ký thất bại. Email có thể đã tồn tại.');
      }
    });
  }
}
