export interface StockResponse {
  ticket_type_id: string;
  total_quantity?: number;
  held_quantity?: number;
  sold_quantity?: number;
  available_quantity: number;
  has_stock: boolean;
}
