export interface TicketType {
  id: string;
  event_id: string;
  sku: string;
  name: string;
  description?: string;
  price: number;
  currency: string;
  max_per_order: number;
  sale_start_at: string;
  sale_end_at: string;
  attributes?: Record<string, any>;
  status: string;
}

export interface Venue {
  id: string;
  organizer_id: string;
  name: string;
  address: string;
  city?: string;
  country: string;
  map_url?: string;
  capacity: number;
  description?: string;
}

export interface Organizer {
  id: string;
  organization_name: string;
  description?: string;
  logo_url?: string;
}

export interface Event {
  id: string;
  organizer_id: string;
  venue_id: string;
  name: string;
  slug: string;
  description?: string;
  poster_url?: string;
  start_at: string;
  end_at: string;
  status: string;
  organizer?: Organizer;
  venue?: Venue;
  ticket_types?: TicketType[];
}

export interface EventQuery {
  page?: number;
  limit?: number;
  query?: string;
  status?: string;
  organizer_id?: string;
  venue_id?: string;
  sort?: string;
  order?: string;
}
