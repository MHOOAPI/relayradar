-- Price snapshots so the change feed can show real movements.
CREATE TABLE IF NOT EXISTS price_history (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT NOT NULL REFERENCES station_groups(id) ON DELETE CASCADE,
    model_id    TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    input_cny   NUMERIC(10, 4),
    output_cny  NUMERIC(10, 4),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS price_history_group_model_idx ON price_history (group_id, model_id, recorded_at DESC);
