export interface CreateOrderItemRequest {
  ticket_type_id: string;
  quantity: number;
}

export interface CreateOrderRequest {
  idempotency_key: string;
  currency?: string;
  items: CreateOrderItemRequest[];
}

export interface OrderItemResponse {
  id: string;
  order_id: string;
  ticket_type_id: string;
  ticket_type_name: string;
  sku: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  created_at: string;
}

export interface OrderResponse {
  id: string;
  user_id: string;
  order_code: string;
  total_amount: number;
  currency: string;
  status: string;
  expires_at?: string;
  paid_at?: string;
  items?: OrderItemResponse[];
  created_at: string;
  updated_at: string;
}
