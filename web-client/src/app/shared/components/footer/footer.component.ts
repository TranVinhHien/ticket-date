import { Component } from '@angular/core';

@Component({
  selector: 'app-footer',
  standalone: true,
  template: `
    <footer class="bg-[#121622] text-[#9BA3B5] py-10 mt-auto border-t border-[#242938]">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex flex-col md:flex-row items-center justify-between gap-4">
          <div class="flex flex-wrap items-center justify-center md:justify-start gap-2.5">
            <div class="w-7 h-7 rounded-md bg-[#1B4BFF] flex items-center justify-center text-white font-display font-bold text-xs">TD</div>
            <span class="font-display text-lg font-bold text-white">TicketDate</span>
            <span class="text-xs text-[#7C8498] sm:ml-2">© 2026 Ticket Date. Nền tảng bán vé sự kiện âm nhạc & concert.</span>
          </div>
          <div class="flex items-center gap-6 text-sm">
            <span class="text-teal-400 text-xs flex items-center gap-2 font-medium">
              <span class="w-2 h-2 rounded-full bg-teal-400"></span>
              Backend Gateway Online (8000)
            </span>
          </div>
        </div>
      </div>
    </footer>
  `
})
export class FooterComponent {}
