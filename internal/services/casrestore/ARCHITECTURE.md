# CAS Restore Architecture

## Two dimensions that must not be mixed

### 1) UploadRoute
Describes which instant-upload / second-pass route is used to restore the file.

- `family` — family route, default
- `person` — person route

This is **not** the same thing as the final folder type.

### 2) DestinationType
Describes where the restored file should finally live.

- `family` — final folder belongs to family cloud
- `person` — final folder belongs to personal cloud

This is **not** the same thing as the upload route.

## Supported combinations

Examples:

- `uploadRoute=family`, `destinationType=family`
- `uploadRoute=family`, `destinationType=person`
- `uploadRoute=person`, `destinationType=person`
- `uploadRoute=person`, `destinationType=family`

Implementation note:

- Product semantics may allow four combinations.
- But cloud-operation implementation must still be backed by the reference flow.
- If a combination has no reference-backed cloud-operation chain yet, it must be treated as unsupported instead of being implemented with guessed SDK substitutions.

## Product defaults

- Default `uploadRoute` = `family`
- `destinationType` should be explicit in requests/UI
- `targetFolderID` only means the final folder ID; it must never be reused to imply upload route semantics

## Coding note

When editing this module, always keep comments around `UploadRoute`, `DestinationType`, and `TargetFolderID`.
They exist specifically to prevent semantic confusion.

## Reference implementation is the only source of truth

For all cloud-disk operations in this module, the implementation target is **reference-flow replication**, not "equivalent behavior".

Reference files:

- `/root/.openclaw/workspace/cloud189-auto-save/src/services/casService.js`
- `/root/.openclaw/workspace/cloud189-auto-save/src/services/cloud189.js`
- `/root/.openclaw/workspace/cloud189-auto-save/src/utils/UploadCryptoUtils.js`

Hard rule:

- command / action names must follow the reference
- endpoint paths must follow the reference
- parameter names and values must follow the reference
- response field extraction order must follow the reference
- signature strategy must follow the reference
- retry conditions and delays must follow the reference
- polling logic must follow the reference
- cleanup order must follow the reference
- family / person route chain must follow the reference

What must not happen:

- do **not** replace the reference flow with a "similar" SDK call just because it looks equivalent
- do **not** rename cloud operations into local abstractions that hide the original command chain
- do **not** simplify protocol steps without explicit evidence from the reference implementation

In short: this module is a Go translation of the working JS reference flow, not an independent redesign.

## Do-not-change alignment checklist

The following aligned items are intentional and must not be casually changed back to SDK substitutions or renamed abstractions:

- upload-domain flow must keep: `getSessionKeyForUpload` -> RSA key cache -> `/person|family/initMultiUpload` -> `/person|family/checkTransSecond` -> `/person|family/commitMultiUploadFile`
- init stage must keep `lazyCheck=1` and must not send md5
- commit stage must keep `fileMd5` + `sliceMd5` + `lazyCheck=1` + `opertype=3`
- commit 403 retry must keep: clear RSA cache -> wait `retry * 2000ms` -> retry
- family -> person must keep AccessToken-signed batch `COPY` + `checkBatchTask` + family cleanup `DELETE`
- family root resolution must keep reference-style lookup instead of hardcoding `-11`
- black-list detection must keep `InfoSecurityErrorCode` / `black list` handling
- commit file-id extraction order must keep reference order
- combinations without a reference-backed cloud-operation chain must remain unsupported

If any future change wants to replace one of the items above, it must first prove the exact reference basis.
