# HOWTO: Google Sheets Authentication and Authorisation


## Steps

1. Create GCP project
2. Configure OAuth2 consent screen
3. Create Oath2 credentials
   - Download credentials and move to /etc/uhppoted/...

4. Enable Google Sheets API
   - Add credentials to API
5. Enable Google Drive API
   - Add credentials to API

## Recovery

1. Delete /var/uhppoted/.../tokens..
2. Run auth procedure again

## Notes
- Mark project as 'Testing' (in OAuth2 settings) to limit access to known users

