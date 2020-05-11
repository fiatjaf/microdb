CREATE TABLE funding (
  time timestamp NOT NULL,
  bucket text NOT NULL,
  msatoshi numeric(13) NOT NULL
);
CREATE INDEX funding_time_idx ON funding (time);
CREATE INDEX funding_bucket_idx ON funding (bucket);

CREATE TABLE data (
  bucket text PRIMARY KEY,
  data jsonb NOT NULL,
  public_write boolean NOT NULL DEFAULT true,
  key text NOT NULL DEFAULT md5(random()::text)
);
CREATE INDEX data_bucket_idx ON data (bucket);

CREATE OR REPLACE FUNCTION bucket_credits (bucket_id text) RETURNS numeric(13) AS $$
  -- 500sat/month = 16.438sat/day
  SELECT (sum(msatoshi) - 16438 * extract('days' from (now() - min(time))))::numeric(13)
  FROM funding WHERE bucket = bucket_id;
$$ LANGUAGE SQL STABLE;
