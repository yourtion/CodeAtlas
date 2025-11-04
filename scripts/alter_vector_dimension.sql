-- Alter vector dimension for existing database
-- Usage: psql -U codeatlas -d codeatlas -v dimension=1536 -f alter_vector_dimension.sql
-- Or: psql -U codeatlas -d codeatlas -c "ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(1536);"

-- Check current dimension
SELECT 
    table_name, 
    column_name,
    udt_name,
    character_maximum_length
FROM information_schema.columns
WHERE table_name = 'vectors' AND column_name = 'embedding';

-- Alter the vector dimension
-- Replace :dimension with your desired dimension (e.g., 768, 1024, 1536, 3072)
ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(:dimension);

-- Verify the change
SELECT 
    table_name, 
    column_name,
    udt_name,
    character_maximum_length
FROM information_schema.columns
WHERE table_name = 'vectors' AND column_name = 'embedding';

-- Note: This will fail if there's existing data with different dimensions
-- To force the change and clear data:
-- TRUNCATE TABLE vectors;
-- ALTER TABLE vectors ALTER COLUMN embedding TYPE vector(:dimension);
