CREATE EXTENSION IF NOT EXISTS postgis;

CREATE INDEX IF NOT EXISTS idx_benches_geo
ON benches
USING GIST (ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography);

ALTER TABLE benches ADD COLUMN IF NOT EXISTS location geography(POINT, 4326);

UPDATE benches
SET location = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
WHERE location IS NULL AND status != 'rejected';

CREATE OR REPLACE FUNCTION update_location()
RETURNS TRIGGER AS $
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_location ON benches;
CREATE TRIGGER trigger_update_location
BEFORE INSERT OR UPDATE ON benches
FOR EACH ROW
EXECUTE FUNCTION update_location();

CREATE INDEX IF NOT EXISTS idx_benches_location
ON benches
USING GIST (location)
WHERE status != 'rejected';
