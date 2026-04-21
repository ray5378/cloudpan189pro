# CAS Restore Architecture

## Upload / restore invariant

For cloud189 CAS restore, **all upload-style restore flows must go through family cloud first**.

Required order:

1. Second-pass / instant restore into **family cloud**
2. Then choose the final target by configuration:
   - transfer/save into **personal cloud**, or
   - keep the restored file in **family cloud**
3. Verify the file actually appears in the selected target folder

## Forbidden path

- Direct person upload / direct person instant-restore as the primary restore path

## Product implication

Backend restore requests must carry a target selector (`person` or `family`).
Frontend can expose this as a switch without changing the family-first invariant.
