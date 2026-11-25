import { Component, input, output, OnInit, OnDestroy, signal } from '@angular/core';

@Component({
  selector: 'app-countdown-timer',
  standalone: true,
  template: `
    <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold tabular-nums"
      [class]="isExpired() ? 'bg-[#FFF1F2] text-[#9F1239] border border-[#FECDD3]' : 'bg-[#FEF3C7] text-[#92400E] border border-[#FDE68A]'">
      <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span>{{ isExpired() ? 'Đã hết hạn giữ vé' : 'Thời gian giữ vé còn lại: ' + timeString() }}</span>
    </div>
  `
})
export class CountdownTimerComponent implements OnInit, OnDestroy {
  readonly expiresAt = input<string | undefined>();
  readonly expired = output<void>();

  readonly timeString = signal<string>('00:00');
  readonly isExpired = signal<boolean>(false);
  private timer: any;

  ngOnInit(): void {
    this.updateRemaining();
    this.timer = setInterval(() => this.updateRemaining(), 1000);
  }

  ngOnDestroy(): void {
    if (this.timer) clearInterval(this.timer);
  }

  private updateRemaining(): void {
    const raw = this.expiresAt();
    if (!raw) {
      this.timeString.set('--:--');
      return;
    }

    const diff = new Date(raw).getTime() - Date.now();
    if (diff <= 0) {
      this.timeString.set('00:00');
      if (!this.isExpired()) {
        this.isExpired.set(true);
        this.expired.emit();
      }
      return;
    }

    const minutes = Math.floor(diff / 60000);
    const seconds = Math.floor((diff % 60000) / 1000);
    this.timeString.set(`${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`);
  }
}
