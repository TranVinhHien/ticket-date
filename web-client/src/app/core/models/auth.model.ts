export interface UserProfile {
  user_id: string;
  full_name?: string;
  avatar_url?: string;
  phone_number?: string;
  home_address?: string;
  current_lat?: number;
  current_lng?: number;
  updated_at: string;
}

export interface UserResponse {
  id: string;
  email: string;
  roles: string[];
  provider: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  profile?: UserProfile;
}

export interface AuthResponse {
  token: string;
  refresh_token: string;
  session_id: string;
  user: UserResponse;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  full_name: string;
  phone_number?: string;
  home_address?: string;
}
