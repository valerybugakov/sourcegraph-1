ALTER TABLE batch_specs
  ADD COLUMN IF NOT EXISTS created_from_raw boolean DEFAULT FALSE NOT NULL;
