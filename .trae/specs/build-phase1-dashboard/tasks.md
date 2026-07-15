# Tasks

- [x] Task 1: Scaffold the dashboard asset and template foundation.
  - [x] Add Bun + Tailwind configuration and wire `make css` to build dashboard CSS
  - [x] Create the base template layout, shared partials, and static asset structure
  - [x] Add dashboard route registration entry points in the API server

- [x] Task 2: Implement session-authenticated dashboard entry flows.
  - [x] Build login page, login submit handler, logout flow, and session cookie handling
  - [x] Build forced password change page and redirect logic for `force_password_change`
  - [x] Add role-based landing redirect behavior for `/`

- [x] Task 3: Implement Phase 1 overview and live map pages.
  - [x] Build company dashboard home for `REFINERY_ADMIN`
  - [x] Build facility landing page for scoped facility roles
  - [x] Add Leaflet map container and JavaScript connection to `/ws/trips/active`

- [x] Task 4: Implement delivery order and trip workflow pages.
  - [x] Build delivery order list with HTMX refresh and queue actions
  - [x] Build delivery order detail page with approval and assignment fragments
  - [x] Build trip list and trip detail pages with timeline, weight bridge, seals, and photos
  - [x] Build weight bridge entry page for trip operations

- [x] Task 5: Implement fleet and station management pages.
  - [x] Build fleet list and vehicle detail pages with maintenance context
  - [x] Build station list and station detail pages with tank levels and delivery history

- [x] Task 6: Implement dashboard user management pages.
  - [x] Build user list and create-user pages
  - [x] Build role assignment UI
  - [x] Build password reset action flow using the existing backend response contract

- [x] Task 7: Add HTMX partial endpoints, template polish, and frontend behavior hardening.
  - [x] Add partial handlers for table refreshes, row swaps, and status fragments
  - [x] Ensure destructive actions use confirmations and async loading states
  - [x] Ensure the map container is not replaced by HTMX swaps

- [x] Task 8: Validate the dashboard end to end.
  - [x] Add focused handler and template tests for login, redirects, and key dashboard pages
  - [x] Verify CSS build, route protection, HTMX flows, and WebSocket map behavior
  - [x] Update `SKILL.md` checklist items that are fully implemented and verified

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1] and [Task 2]
- [Task 4] depends on [Task 1] and [Task 2]
- [Task 5] depends on [Task 1] and [Task 2]
- [Task 6] depends on [Task 1] and [Task 2]
- [Task 7] depends on [Task 3], [Task 4], [Task 5], and [Task 6]
- [Task 8] depends on [Task 3], [Task 4], [Task 5], [Task 6], and [Task 7]
