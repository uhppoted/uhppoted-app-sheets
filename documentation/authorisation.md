# HOWTO: Google Sheets Authentication and Authorisation

---
#### NOTE

__The terminal implementation of Google's OAuth2 flow was deprecated completely on 3 October 2022 (see [OOB Migration](https://developers.google.com/identity/protocols/oauth2/resources/oob-migration)).__

*_uhppoted-app-sheets_ has been updated for the new _OAuth2_ flow but the changes are only in the current development version. To build the development version please see [Building from source](https://github.com/uhppoted/uhppoted-app-sheets#building-from-source)
in the [README](https://github.com/uhppoted/uhppoted-app-sheets)*.

---

This _HOWTO_ describes the (somewhat) tortuous process of giving _uhppoted-app-sheets_ access to a _Google Sheets_ spreadsheet. In broad outline, it describes the steps to create:

- a _placeholder_ Google Cloud project (GCP) to identify your specific instance of _uhppoted-app-sheets_. These are
  free (up to a reasonable limit) and only require a Google account.

  The GCP project requires
   - a configured OAuth2 consent screen for the project
   - the Google Sheets API to be enabled
   - the Google Drive API to be enabled
   
- a set of GCP credentials for your instance of _uhppoted-app-sheets_
- a set of authorisation tokens for _Google Sheets_ and _Google Drive_

Most of this is fairly straightforward but a little intimidating if you haven't ever done it before. It also (typically)
only has to be done once, on project setup.

## Google Cloud Project

To create a Google Cloud project, log in to your _Google_ account and:

1. Open the [Google Cloud console](https://console.cloud.google.com/home)
2. Create a [new project](https://console.cloud.google.com/projectcreate)
   - Choose a suitable name e.g. _uhppoted-app-sheets-hogwarts_
   - Choose an organisation if you have one, otherwise leave the _Location_ as _No organisation_
   - Click _Create_
3. Check that you have the project selected in the dropdown list (at the top of the page) and open 
   the project dashboard.

The next step is to enable the _Google Sheets_ and _Google Drive_ APIs:
1. Open the [_API Library_](https://console.cloud.google.com/apis/library) page
2. Search for _Google Sheets_ and enable the _Google Sheets API_
3. Open the [_API Library_](https://console.cloud.google.com/apis/library) page again
4. Search for _Google Drive_ and enable the _Google Drive API_

Once the APIs are enabled, it's time to configure _OAuth2 consent screen_:
1. Open the [_OAuth consent screen_](https://console.cloud.google.com/apis/credentials/consent) page
2. Choose _External_ (unless you initially set this project up as an organisation project)
3. Click _Create_
4. Fill in the requested information:
    - _App name_: `uhppoted-app-sheets-hogwarts`
    - _User support email_: _\<your email address\>_
    - _App logo_: (ignore)
    - _Developer contact information_: _\<your email address\>_
5. Click _Save and continue_
6. Ignore the _Scopes_ page and click _Save and continue_
7. On the _Test users_ page:
    - Add the email addresses of the account that is going to authorise access to the spreadsheet
    - Click _Save and continue_
8. You're done - click _Back to dashboard_

Now create a set of _OAuth2_ credentials:
1. Open the [_Credentials_](https://console.cloud.google.com/apis/credentials) page
2. Click _Create credentials_ and choose _OAuth Client ID_:
    - _Application type_: `Desktop app`
    - _Name_: `uhppoted-app-sheets-hogwarts`
    - Click _Create_ and download the JSON file from the popup
    - You're done here, click _Ok_
3. Copy the downloaded credentials file to the uhppoted-app-sheets _etc_ folder (or a convenient folder 
   of your choice). The default `credentials` are expected to be:

| Platform | Credentials                                                          |
|----------|----------------------------------------------------------------------|
| Linux    | `/etc/uhppoted/sheets/.google/credentials.json`                      |
| MacOS    | `/usr/local/etc/com.github.uhppoted/sheets/.google/credentials.json` |
| Windows  | `\Program Data\uhppoted\sheets\.google\credentials.json`             |

## _uhppoted-app-sheets_ authorisation

Installation for _uhppoted-app-sheets_ is described in the [README](https://github.com/uhppoted/uhppoted-app-sheets#installation)
but for the current _development_ build you will have to follow the steps under [Building from source](https://github.com/uhppoted/uhppoted-app-sheets#building-from-source).

Having installed (or built) _uhppoted-app-sheets_, the next step is to _authorise_ access to the _Google Sheets_ spreadsheet:

1. Run the `authorise` command:
```
uhppoted-app-sheets authorise --url <spreadsheet>

Where <spreadsheet> is a Google Docs URL:

   https://docs.google.com/spreadsheets/d/<spreadsheet ID>

e.g. https://docs.google.com/spreadsheets/d/1_erZMyFmO6PM0PrAfEqdsiH9haiw-2UqY0kLwo_WTO8

The spreadsheet ID can be copied from the URL in the browser.
```

2. Open [http://localhost/auth.html](http://localhost/auth.html) in your browser (the `authorise` command should open this automatically but systems vary wildly so you may need to open it manually).

3. Follow both the _Google Sheets_ and the _Google Drive_ links provided to authorise access to the spreadsheet data and version information.

4. On completion of the above you should have a working set of credentials and authorisation tokens for _uhppoted-app-sheets_ on your system. 
 
   If the system on which you authorised _uhppoted-app-sheets_ is not the target system (e.g. a headless _Raspberry Pi_) you will need to copy the credentials and token files to the target system:

| Platform | File                 | Destination                                                 |
|----------|----------------------|-------------------------------------------------------------|
| Linux    | `credentials.json`   | `/etc/uhppoted/sheets/.google/credentials.json`             |
|          | `credentials.sheets` | `/var/uhppoted/sheets/.google/credentials.sheets`           |
|          | `credentials.drive`  | `/var/uhppoted/sheets/.google/credentials.drive`            |
|          |                      |                                                             |
| MacOS    | `credentials.json`   | `/usr/local/etc/uhppoted/sheets/.google/credentials.json`   |
|          | `credentials.sheets` | `/usr/local/var/uhppoted/sheets/.google/credentials.sheets` |
|          | `credentials.drive`  | `/usr/local/var/uhppoted/sheets/.google/credentials.drive`  |
|          |                      |                                                             |
| Windows  | `credentials.json`   | `\Program Data\uhppoted\sheets\.google/credentials.json`    |
|          | `credentials.sheets` | `\Program Data\uhppoted\sheets\.google/credentials.sheets`  |
|          | `credentials.drive`  | `\Program Data\uhppoted\sheets\.google/credentials.drive`   |

## Reference Documentation

1. [Google Sheets API: Setup the sample]https://developers.google.com/sheets/api/quickstart/go#set_up_the_sample
2. [Google Identity: Using OAuth 2.0 to Access Google APIs](https://developers.google.com/identity/protocols/oauth2)
3. [Google Workspace: Create access credentials](https://developers.google.com/workspace/guides/create-credentials#desktop-app)
4. [Troubleshoot authentication & authorization issues](https://developers.google.com/sheets/api/troubleshoot-authentication-authorization)
