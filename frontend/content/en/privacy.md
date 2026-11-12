---
title: Privacy policy
description: How Saku no Ie (host-occupied simple lodging) handles personal information (draft, pending legal review).
version: "2026-11-12"
effectiveDate: "2026-11-12"
placeholder: true
todo: "要差し替え: draft pending legal review. Fill in every 【要記入】 (operator name/representative, contact, hosting provider, retention periods) and resolve every 【要法務確認】 before removing placeholder. version must equal privacy_version in backend/internal/services/policy_versions.json."
---

> **This policy is a draft. It needs legal review before publication.** Items marked 【要記入】 (to be filled in) or 【要法務確認】 (legal review needed) are not final.
>
> The Japanese version of this policy is authoritative. This English translation is provided for reference only.

Saku no Ie (the "Facility") complies with the Act on the Protection of Personal Information (APPI) and other applicable laws and regulations, and handles guests' personal information as follows.

## 1. Operator

- Name: Saku no Ie (host-occupied simple lodging, 家主居住型 簡易宿所)
- Address: 1412-1 Yasuhara, Saku, Nagano, Japan
- License under the Inns and Hotels Act: Nagano Prefecture Saku Health Center Directive 07 Sa-Ho No. 11-68 (長野県佐久保健所指令07佐保第11-68号)
- Operator's name (representative): 【要記入】
- Contact (email address and phone number): 【要記入】

## 2. Information we collect

### 2.1 Guest registration (guest register)

To keep the guest register required by the Inns and Hotels Act, the guest registration form on this website collects:

- About your stay: check-in and check-out dates, expected arrival time, planned number of guests
- For each guest: name, name in kana (optional), address, phone number (optional), email address (optional), whether the guest has an address in Japan
- For guests without an address in Japan: nationality and passport number
- Booking source (such as Airbnb) and booking reference (optional)

The presentation of passports and the keeping of copies under Article 8 of the terms of accommodation are handled in person by the host on arrival. This website never collects passport images. How and for how long copies are kept: 【要記入】.

### 2.2 Records created by the process

- Consent records: the versions of the terms of accommodation and of this policy that you accepted, the time of each acceptance, and the time of registration
- Identity verification records: the time at which the host verified identity in person on arrival, and the administrator who did it
- Edit link: a link for adding companions and making corrections after registration. We never store the secret value in the link itself, only its hash (SHA-256), its expiry and the time it was last used
- Activity records (audit log): the type and time of operations such as registration, correction and export, the type of actor, and the names of the changed fields. Field values (names, passport numbers and so on) are not recorded
- Duplicate-submission prevention: so that the same registration is not created twice, a hash of each submission's identifier and the identifier of the created registration are kept for 24 hours and then deleted

### 2.3 Post-stay survey

The survey is anonymous. We store only your answers, the facility, the survey version and the day of the answer (day only; no time), and never link them to a registration, booking or name. Please do not write anything that identifies you in the free-text fields.

### 2.4 Access information

- IP address: used only temporarily in the server's memory to prevent excessive access (rate limiting). It is not stored in the database or in logs
- Server logs: request ID, time, method, route, result and processing time. Logs contain no personal information such as names, passport numbers, edit links or IP addresses
- Traffic passes through Cloudflare's network (section 6). Information Cloudflare collects is handled under its own privacy policy

### 2.5 Cookies and browser storage

- The Facility uses no cookies, tags or tracking technologies for analytics or advertising
- A cookie (`NEXT_LOCALE`) may be used to remember your display language
- The value of an edit link is kept temporarily only in the browser tab where you opened it (sessionStorage) and is gone when the tab is closed
- Cloudflare may set cookies to keep traffic secure

### 2.6 Administrator information

Sign-in to the admin console uses Firebase Authentication (provided by Google LLC). It applies only to administrators, such as the host, who have been granted access. No guest information is sent to Firebase.

## 3. Purposes of use

1. Keeping the guest register under the Inns and Hotels Act
1. Accepting stays, verifying identity on arrival, and contacting you about your stay
1. Issuing and managing edit links for adding to or correcting a registration
1. Responding to administrative agencies, the police and others as required by law
1. Operating the system securely, including preventing unauthorized access
1. Improving our service through survey statistics

We do not use personal information beyond these purposes.

## 4. Retention periods

- Guest register: kept for 3 years under the Inns and Hotels Act. 【要法務確認: legal basis, starting date and exceptions】
- Registrations for stays that did not take place: 【要記入】【要法務確認】
- Activity records (audit log): 【要記入】
- Edit links: valid until the end of the check-out day (facility local time). How long records are kept after expiry: 【要記入】
- Duplicate-submission records: deleted after 24 hours
- Anonymous survey: kept for statistics. Retention period: 【要記入】
- Backups: 【要記入】

Information whose retention period has ended is deleted unless the law requires it to be kept. The deletion procedure is being prepared. 【要記入】

## 5. Provision to third parties

The Facility does not provide personal information to third parties without your consent, except in the following cases (APPI Article 27, paragraph 1):

- When required by law (for example, an inquiry from the police based on law, or a request from a public health center under the Inns and Hotels Act)
- When it is necessary to protect a person's life, body or property and it is difficult to obtain your consent
- Other cases permitted by law

## 6. External services (entrustment, etc.)

This website uses the following external services. 【要法務確認: whether each counts as "entrustment" under the APPI】

- Hosting provider (servers and database): 【要記入】 (not yet decided)
- Cloudflare, Inc. (United States): content delivery (CDN), relaying traffic to our servers (Cloudflare Tunnel), and traffic security
- Google LLC (United States): Firebase Authentication (administrator sign-in only)

Bookings and payments are made on Airbnb. Personal information relating to bookings and payments is handled by Airbnb under Airbnb's privacy policy. This website is not connected to Airbnb and does not import booking information automatically (the booking reference is optional and entered by you).

## 7. Use of businesses in foreign countries

Cloudflare, Inc. and Google LLC in section 6 are businesses in the United States, and their facilities may be outside Japan. The United States has no comprehensive federal personal-information protection law; protection is provided by state laws and sector-specific laws. For each company's measures to protect personal information, please see its privacy policy. 【要法務確認: the information to be provided under APPI Article 28, and whether consent or equivalent measures are needed】

## 8. Security measures

- All traffic is encrypted (TLS)
- Passport numbers are encrypted by the application (AES-256-GCM) before they are stored, and the encryption key is managed separately from the database. The admin console and CSV export show only the last 4 digits
- Addresses, phone numbers and email addresses are masked in the admin console, and revealing them is recorded
- Only signed-in administrators who have been granted access to the facility can use the admin console
- For edit links, only a hash of the secret value is stored, and the links expire
- Logs contain no personal information. Access is rate-limited to prevent misuse
- Organizational, human and physical security measures: 【要記入】

## 9. Requests for disclosure, etc.

You or your representative may request notification of the purposes of use, disclosure, correction, addition or deletion, suspension of use or erasure, suspension of provision to third parties, and disclosure of records of provision to third parties, for personal information the Facility holds. Please contact us (【要記入】). We will confirm your identity and respond as required by law. Fees, if any: 【要記入】.

The guest register that must be kept under the Inns and Hotels Act may not be deletable during its retention period. You can add companions and correct some information yourself through the edit link until the end of the check-out day.

## 10. Changes to this policy

The Facility may revise this policy in response to changes in laws or in its services. The revised policy will be posted on this page with its version and effective date. When you register, the version you accepted is recorded.

## 11. Contact

Inquiries about the handling of personal information: 【要記入】 (email address and phone number)

---

Version: 2026-11-12 (draft)\
Effective date: 【要記入】 (to be set after legal review)
