import { TestBed } from '@angular/core/testing';
import { CheckoutStateService } from './checkout-state.service';
import { Event, TicketType } from '../models/event.model';

describe('CheckoutStateService', () => {
  let service: CheckoutStateService;

  const mockEvent: Event = {
    id: 'e1',
    organizer_id: 'o1',
    venue_id: 'v1',
    name: 'Concert 2026',
    slug: 'concert-2026',
    start_at: new Date().toISOString(),
    end_at: new Date().toISOString(),
    status: 'PUBLISHED'
  };

  const mockTicketType: TicketType = {
    id: 't1',
    event_id: 'e1',
    sku: 'VIP-01',
    name: 'VIP Ticket',
    price: 500000,
    currency: 'VND',
    max_per_order: 4,
    sale_start_at: new Date().toISOString(),
    sale_end_at: new Date().toISOString(),
    status: 'ACTIVE'
  };

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(CheckoutStateService);
  });

  it('should be initially empty', () => {
    expect(service.event()).toBeNull();
    expect(service.items().length).toBe(0);
    expect(service.totalAmount()).toBe(0);
    expect(service.totalQuantity()).toBe(0);
    expect(service.idempotencyKey()).toBe('');
  });

  it('should set checkout selection and calculate totals and generate idempotency key', () => {
    service.setCheckoutSelection(mockEvent, [
      { ticketType: mockTicketType, quantity: 2 }
    ]);

    expect(service.event()?.name).toBe('Concert 2026');
    expect(service.items().length).toBe(1);
    expect(service.totalQuantity()).toBe(2);
    expect(service.totalAmount()).toBe(1000000);
    expect(service.idempotencyKey()).toBeTruthy();
    expect(service.idempotencyKey().length).toBeGreaterThan(10);
  });

  it('should clear selection cleanly', () => {
    service.setCheckoutSelection(mockEvent, [
      { ticketType: mockTicketType, quantity: 2 }
    ]);
    expect(service.totalAmount()).toBe(1000000);

    service.clear();
    expect(service.event()).toBeNull();
    expect(service.items().length).toBe(0);
    expect(service.totalAmount()).toBe(0);
    expect(service.idempotencyKey()).toBe('');
  });
});
