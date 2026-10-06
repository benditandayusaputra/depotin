-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE depots (
  id uuid PRIMARY KEY,
  name text NOT NULL CHECK (char_length(name) BETWEEN 3 AND 80),
  slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
  phone text NOT NULL,
  address text NOT NULL DEFAULT '',
  lat double precision,
  lng double precision,
  timezone text NOT NULL DEFAULT 'Asia/Jakarta',
  open_time time NOT NULL DEFAULT '07:00',
  close_time time NOT NULL DEFAULT '20:00',
  delivery_fee bigint NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),
  is_accepting_orders boolean NOT NULL DEFAULT true,
  auto_confirm_known boolean NOT NULL DEFAULT true,
  loyalty_every int CHECK (loyalty_every IS NULL OR loyalty_every BETWEEN 2 AND 50),
  reminder_lead_days int NOT NULL DEFAULT 1 CHECK (reminder_lead_days BETWEEN 0 AND 7),
  default_days_per_gallon numeric(5,2) NOT NULL DEFAULT 4.00 CHECK (default_days_per_gallon BETWEEN 0.5 AND 30),
  is_demo boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  role text NOT NULL CHECK (role IN ('owner', 'courier')),
  name text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 80),
  phone text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  is_active boolean NOT NULL DEFAULT true,
  failed_login_count int NOT NULL DEFAULT 0,
  locked_until timestamptz,
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX users_depot_idx ON users (depot_id);

CREATE TABLE sessions (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users(id),
  family_id uuid NOT NULL,
  token_hash bytea NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  rotated_at timestamptz,
  revoked_at timestamptz,
  user_agent text NOT NULL DEFAULT '',
  ip inet,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_active_idx ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX sessions_family_idx ON sessions (family_id);

CREATE TABLE products (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  name text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 60),
  kind text NOT NULL CHECK (kind IN ('refill', 'new_gallon', 'other')),
  price bigint NOT NULL CHECK (price >= 0),
  is_active boolean NOT NULL DEFAULT true,
  sort_order int NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (depot_id, name)
);

CREATE TABLE customers (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
  phone text NOT NULL,
  address text NOT NULL DEFAULT '',
  address_note text NOT NULL DEFAULT '',
  area text NOT NULL DEFAULT '',
  lat double precision,
  lng double precision,
  token_hash bytea UNIQUE,
  token_enc bytea,
  token_rotated_at timestamptz,
  source text NOT NULL CHECK (source IN ('owner', 'public')),
  is_verified boolean NOT NULL DEFAULT false,
  usual_qty int NOT NULL DEFAULT 1 CHECK (usual_qty BETWEEN 1 AND 50),
  loan_balance int NOT NULL DEFAULT 0 CHECK (loan_balance >= 0),
  stamp_count int NOT NULL DEFAULT 0 CHECK (stamp_count >= 0),
  days_per_gallon numeric(6,2),
  prediction_samples int NOT NULL DEFAULT 0,
  prediction_confidence text NOT NULL DEFAULT 'none' CHECK (prediction_confidence IN ('none', 'low', 'medium', 'high')),
  last_delivered_at timestamptz,
  last_delivered_qty int,
  predicted_empty_at timestamptz,
  reminder_snoozed_until date,
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (depot_id, phone)
);
CREATE INDEX customers_reminder_idx ON customers (depot_id, predicted_empty_at) WHERE is_active AND is_verified;
CREATE INDEX customers_name_trgm_idx ON customers USING gin (name gin_trgm_ops);
CREATE INDEX customers_phone_trgm_idx ON customers USING gin (phone gin_trgm_ops);
CREATE INDEX customers_address_trgm_idx ON customers USING gin (address gin_trgm_ops);
CREATE INDEX customers_depot_created_idx ON customers (depot_id, created_at DESC);

CREATE TABLE reminders (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  customer_id uuid NOT NULL REFERENCES customers(id),
  due_date date NOT NULL,
  predicted_empty_at timestamptz NOT NULL,
  status text NOT NULL CHECK (status IN ('queued', 'sent', 'ordered', 'skipped', 'expired')),
  sent_at timestamptz,
  sent_by uuid REFERENCES users(id),
  order_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (customer_id, due_date)
);
CREATE INDEX reminders_depot_status_due_idx ON reminders (depot_id, status, due_date);

CREATE TABLE orders (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  customer_id uuid NOT NULL REFERENCES customers(id),
  code text NOT NULL,
  source text NOT NULL CHECK (source IN ('public', 'link', 'reminder', 'owner', 'courier')),
  status text NOT NULL CHECK (status IN ('pending', 'confirmed', 'on_delivery', 'delivered', 'cancelled')),
  fulfilment text NOT NULL CHECK (fulfilment IN ('delivery', 'pickup')),
  scheduled_date date NOT NULL,
  delivery_name text NOT NULL,
  delivery_phone text NOT NULL,
  delivery_address text NOT NULL DEFAULT '',
  delivery_note text NOT NULL DEFAULT '',
  note text NOT NULL DEFAULT '',
  refill_qty int NOT NULL DEFAULT 0 CHECK (refill_qty >= 0),
  free_qty int NOT NULL DEFAULT 0 CHECK (free_qty >= 0),
  subtotal bigint NOT NULL CHECK (subtotal >= 0),
  delivery_fee bigint NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),
  discount bigint NOT NULL DEFAULT 0 CHECK (discount >= 0),
  total bigint NOT NULL CHECK (total >= 0),
  payment_method text CHECK (payment_method IS NULL OR payment_method IN ('cash', 'transfer', 'qris')),
  payment_status text NOT NULL DEFAULT 'unpaid' CHECK (payment_status IN ('unpaid', 'paid')),
  paid_at timestamptz,
  courier_id uuid REFERENCES users(id),
  gallons_returned int CHECK (gallons_returned IS NULL OR gallons_returned >= 0),
  reminder_id uuid REFERENCES reminders(id),
  track_token_hash bytea NOT NULL UNIQUE,
  idempotency_key text,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  confirmed_at timestamptz,
  dispatched_at timestamptz,
  delivered_at timestamptz,
  cancelled_at timestamptz,
  cancel_reason text,
  UNIQUE (depot_id, code)
);
CREATE UNIQUE INDEX orders_idempotency_idx ON orders (depot_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX orders_board_idx ON orders (depot_id, status, scheduled_date);
CREATE INDEX orders_recent_idx ON orders (depot_id, created_at DESC);
CREATE INDEX orders_customer_delivered_idx ON orders (customer_id, delivered_at DESC) WHERE status = 'delivered';
CREATE INDEX orders_courier_queue_idx ON orders (courier_id, status) WHERE status IN ('confirmed', 'on_delivery');
CREATE INDEX orders_customer_active_idx ON orders (customer_id) WHERE status IN ('pending', 'confirmed', 'on_delivery');

ALTER TABLE reminders ADD CONSTRAINT reminders_order_fk FOREIGN KEY (order_id) REFERENCES orders(id);

CREATE TABLE order_items (
  id uuid PRIMARY KEY,
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id uuid NOT NULL REFERENCES products(id),
  product_name text NOT NULL,
  product_kind text NOT NULL,
  unit_price bigint NOT NULL CHECK (unit_price >= 0),
  qty int NOT NULL CHECK (qty BETWEEN 1 AND 50),
  line_total bigint NOT NULL CHECK (line_total >= 0)
);
CREATE INDEX order_items_order_idx ON order_items (order_id);

CREATE TABLE order_events (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  order_id uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  type text NOT NULL CHECK (type IN ('created', 'confirmed', 'assigned', 'dispatched', 'delivered', 'cancelled', 'paid')),
  actor_type text NOT NULL CHECK (actor_type IN ('user', 'customer', 'system')),
  actor_id uuid,
  idempotency_key text,
  meta jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX order_events_idempotency_idx ON order_events (order_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX order_events_order_idx ON order_events (order_id, created_at);

CREATE TABLE gallon_ledger (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  customer_id uuid NOT NULL REFERENCES customers(id),
  order_id uuid REFERENCES orders(id),
  kind text NOT NULL CHECK (kind IN ('delivery', 'adjustment', 'lost')),
  delta int NOT NULL CHECK (delta <> 0),
  balance_after int NOT NULL CHECK (balance_after >= 0),
  note text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (kind = 'delivery' OR char_length(note) > 0)
);
CREATE INDEX gallon_ledger_customer_idx ON gallon_ledger (customer_id, created_at DESC);

CREATE TABLE order_counters (
  depot_id uuid NOT NULL REFERENCES depots(id),
  day date NOT NULL,
  last_no int NOT NULL,
  PRIMARY KEY (depot_id, day)
);

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY,
  depot_id uuid NOT NULL REFERENCES depots(id),
  user_id uuid REFERENCES users(id),
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id uuid,
  meta jsonb NOT NULL DEFAULT '{}'::jsonb,
  ip inet,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_depot_idx ON audit_logs (depot_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS order_counters;
DROP TABLE IF EXISTS gallon_ledger;
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS order_items;
ALTER TABLE reminders DROP CONSTRAINT IF EXISTS reminders_order_fk;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS depots;
