import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./features/home/home.page').then(m => m.HomePageComponent)
  },
  {
    path: 'events/:slug',
    loadComponent: () => import('./features/event-detail/event-detail.page').then(m => m.EventDetailPageComponent)
  },
  {
    path: 'login',
    loadComponent: () => import('./features/auth/login.page').then(m => m.LoginPageComponent)
  },
  {
    path: 'checkout',
    loadComponent: () => import('./features/checkout/checkout.page').then(m => m.CheckoutPageComponent),
    canActivate: [authGuard]
  },
  {
    path: 'booking-result/:orderId',
    loadComponent: () => import('./features/booking-result/booking-result.page').then(m => m.BookingResultPageComponent),
    canActivate: [authGuard]
  },
  {
    path: 'my-tickets',
    loadComponent: () => import('./features/my-tickets/my-tickets.page').then(m => m.MyTicketsPageComponent),
    canActivate: [authGuard]
  },
  {
    path: 'my-tickets/:orderId',
    loadComponent: () => import('./features/my-tickets/ticket-detail.page').then(m => m.TicketDetailPageComponent),
    canActivate: [authGuard]
  },
  {
    path: '**',
    redirectTo: ''
  }
];
