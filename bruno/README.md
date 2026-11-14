# Ticket Bruno API Collection

API testing collection for the ticket microservices architecture using the Bruno API Client.

## Directory Structure

- `auth/`: Registration, login, token refresh (User Service).
- `user/`: User profile management (User Service).
- `media/`: Media asset & avatar uploads (User Service).
## Usage Guide

1. Open this directory in the **Bruno** application.
2. Select Environment: **Local**.
3. Configure environment variables (`environments/Local.bru`):
   - `base_url`: `http://localhost:8000` (API Gateway URL).
   - `token`: Bearer access token (obtained from login request in `auth` directory).
