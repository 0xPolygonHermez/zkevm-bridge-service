-- +migrate Down

DROP FUNCTION IF EXISTS mt.get_nodes(BIGINT, BYTEA);

-- +migrate Up

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION mt.get_nodes(index BIGINT, root BYTEA) 
    RETURNS TABLE (
        key_hash BYTEA,
        left_hash BYTEA,
        right_hash BYTEA
    )
    LANGUAGE plpgsql
AS $$
DECLARE
	rt RECORD;
BEGIN

    FOR i IN reverse 31..0
    LOOP
        SELECT * INTO rt FROM mt.rht WHERE key = root;
        key_hash := rt.key;
        left_hash := rt.value[1];
        right_hash := rt.value[2];
        RETURN NEXT;

        IF index & (1 << i) > 0 THEN
            root := right_hash;
        ELSE
            root := left_hash;
        END IF;
    END LOOP;
END;$$;
-- +migrate StatementEnd