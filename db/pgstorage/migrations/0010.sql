-- +migrate Up

-- This migration will delete all empty blocks
DELETE FROM sync.block
WHERE NOT EXISTS (SELECT *
        FROM sync.claim 
        WHERE sync.claim.block_id = sync.block.id)
    AND NOT EXISTS (SELECT *
        FROM sync.deposit  
        WHERE sync.deposit.block_id = sync.block.id)
    AND NOT EXISTS (SELECT *
        FROM sync.token_wrapped 
        WHERE sync.token_wrapped.block_id = sync.block.id)
    AND NOT EXISTS (SELECT *
        FROM sync.exit_root 
        WHERE sync.exit_root.block_id = sync.block.id)
    AND NOT EXISTS (SELECT *
        FROM mt.rollup_exit 
        WHERE mt.rollup_exit.block_id = sync.block.id);   


-- +migrate Down

-- no action is needed, the data must remain deleted as it is useless