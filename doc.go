// Copyright 2023 uhppoted@twyst.co.za. All rights reserved.
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

/*
Package uhppoted-app-sheets integrates the uhppote-core API with access control lists stored as Google Sheets.

uhppoted-app-sheets can be used from the command line but is really intended to be run from a cron job to maintain
the cards and permissions on a set of access controllers from a unified access control list (ACL).

uhppoted-app-s3 supports the following commands:

  - authorise, to authorise application access to the Google Sheets worksheet
  - load-acl, to download an ACL from a Google Sheets worksheet to a set of access controllers
  - upload-acl, to retrieve the ACL from a set of controllers and write it to a Google Sheets worksheet
  - compare-acl, to compare an ACL from a Google Sheets worksheet with the cards and permissons on a set of access controllers
  - get, to download a Google Sheets worksheet as a TSV file
  - put, to store a TSV file to a Google Sheets worksheet
*/
package sheets
