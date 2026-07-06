# Changelog

## v1.0.0.26070601

- Added VeloSSH version display in the server manager.
- Added update checking with update, cancel, and skip-this-version choices.
- Added automatic update download and installation with an in-app progress modal.
- Added a setting to disable update checks.
- Added release workflow support for publishing platform builds from changelog release notes.
- Improved SSH shell exit handling so `exit` returns to the server list.
- Added real `vssh export` and `vssh import` configuration backup commands with optional secret backup.
- Added `.vsshignore` filtering for recursive local folder uploads.
