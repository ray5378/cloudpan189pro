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

## Product defaults

- Default `uploadRoute` = `family`
- `destinationType` should be explicit in requests/UI
- `targetFolderID` only means the final folder ID; it must never be reused to imply upload route semantics

## Coding note

When editing this module, always keep comments around `UploadRoute`, `DestinationType`, and `TargetFolderID`.
They exist specifically to prevent semantic confusion.
