import { Injectable, signal, computed } from '@angular/core';
import { Event, TicketType } from '../models/event.model';

export interface SelectedTicketItem {
  ticketType: TicketType;
  quantity: number;
}

@Injectable({
  providedIn: 'root'
})
export class CheckoutStateService {
  private readonly _event = signal<Event | null>(null);
  private readonly _items = signal<SelectedTicketItem[]>([]);
  private readonly _idempotencyKey = signal<string>('');

  readonly event = this._event.asReadonly();
  readonly items = this._items.asReadonly();
  readonly idempotencyKey = this._idempotencyKey.asReadonly();

  readonly totalAmount = computed(() => {
    return this._items().reduce((sum, item) => sum + (item.ticketType.price * item.quantity), 0);
  });

  readonly totalQuantity = computed(() => {
    return this._items().reduce((sum, item) => sum + item.quantity, 0);
  });

  setCheckoutSelection(event: Event, items: SelectedTicketItem[]): void {
    this._event.set(event);
    this._items.set(items);
    // Generate fresh idempotency key for this checkout session
    this._idempotencyKey.set(this.generateUUID());
  }

  regenerateIdempotencyKey(): void {
    this._idempotencyKey.set(this.generateUUID());
  }

  clear(): void {
    this._event.set(null);
    this._items.set([]);
    this._idempotencyKey.set('');
  }

  private generateUUID(): string {
    if (typeof crypto !== 'undefined' && crypto.randomUUID) {
      return crypto.randomUUID();
    }
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
      const r = (Math.random() * 16) | 0;
      const v = c === 'x' ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  }
}
