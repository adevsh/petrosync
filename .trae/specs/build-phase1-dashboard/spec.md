# Phase 1 Dashboard Spec

## Why
Phase 1 backend flow is largely in place, but operators still lack the dashboard needed to run the trip lifecycle from the web UI. We need a minimal, production-usable dashboard that matches the existing Gin, HTMX, session-auth, and WebSocket architecture already described in `SKILL.md`.

## What Changes
- Add a Bun + Tailwind dashboard frontend pipeline for server-rendered pages
- Add dashboard routes, templates, partials, and handlers for the Phase 1 page map
- Add session-authenticated dashboard pages for login and forced password change
- Add live operational views for delivery orders, trips, weight bridge, fleet, stations, and users
- Add HTMX partial endpoints for queue refresh, approval, assignment, and page fragments
- Add Leaflet-based live map integration using the existing `/ws/trips/active` WebSocket feed
- Reuse existing API and service behavior instead of duplicating business logic in templates

## Impact
- Affected specs: Phase 1 Dashboard Tasks, Dashboard Architecture, Real-time WebSocket Architecture
- Affected code: `cmd/api/main.go`, `internal/middleware/session.go`, `internal/handler/*`, `internal/ws/*`, `templates/`, `static/`, `Makefile`, Bun/Tailwind assets

## ADDED Requirements
### Requirement: Dashboard Asset Pipeline
The system SHALL provide a Bun + Tailwind build pipeline for dashboard assets that can build and watch CSS used by server-rendered dashboard pages.

#### Scenario: CSS build
- **WHEN** a developer runs the dashboard CSS build command
- **THEN** compiled CSS is written to the dashboard static asset location

### Requirement: Session-authenticated Dashboard
The system SHALL provide session-authenticated dashboard pages for web users, separate from mobile JWT flows.

#### Scenario: Login success
- **WHEN** a valid dashboard user submits username and password
- **THEN** the system creates a session and redirects the user to the correct landing page for their role

#### Scenario: Forced password change
- **WHEN** a user with `force_password_change = true` logs in
- **THEN** the system redirects them to a forced password change page before allowing normal dashboard navigation

### Requirement: Base Dashboard Layout
The system SHALL provide a shared dashboard layout with sidebar, topbar, breadcrumb support, and HTMX-safe content regions.

#### Scenario: Shared navigation
- **WHEN** a user opens any dashboard page
- **THEN** the page renders inside the shared layout with role-appropriate navigation links

### Requirement: Operations Overview Pages
The system SHALL provide the minimum Phase 1 overview pages defined in `SKILL.md`, including company, facility, delivery order, and trip views.

#### Scenario: Company dashboard
- **WHEN** a refinery admin opens `/`
- **THEN** the page shows company-wide operational summary data and a live trip map container

#### Scenario: Scoped landing redirect
- **WHEN** a non-refinery-admin opens `/`
- **THEN** the system redirects them to the scoped facility, station, or fleet landing page defined by their role

### Requirement: Delivery Order Workflow UI
The system SHALL provide delivery order list and detail pages with HTMX actions for approval and assignment using existing backend rules.

#### Scenario: Approve a delivery order
- **WHEN** an authorized user approves a delivery order from the queue or detail page
- **THEN** the row or detail fragment updates in place without a full page reload

### Requirement: Trip Workflow UI
The system SHALL provide trip list and trip detail pages that expose the Phase 1 timeline, weight bridge state, seals, and photo evidence.

#### Scenario: View trip state
- **WHEN** an operator opens a trip detail page
- **THEN** the page shows the current state, event timeline, weight bridge information, seals, and photo gallery data

### Requirement: Live Trip Map
The system SHALL provide a Leaflet map page or region that listens to `/ws/trips/active` and updates trip markers in real time.

#### Scenario: Live map update
- **WHEN** a new trip location message arrives over WebSocket
- **THEN** the matching map marker updates without HTMX replacing the map container

### Requirement: Fleet and Station Pages
The system SHALL provide list and detail pages for vehicles and stations, including maintenance and tank-level context needed in Phase 1.

#### Scenario: Station detail
- **WHEN** a station manager opens a station detail page
- **THEN** the page shows tank information and delivery history relevant to that station

### Requirement: User Management Pages
The system SHALL provide dashboard pages for user list, create, role assignment, and password reset using existing admin APIs and RBAC rules.

#### Scenario: Reset a password from the dashboard
- **WHEN** an authorized admin resets a user password from the UI
- **THEN** the dashboard shows the success state and any fallback temporary password data returned by the backend

## MODIFIED Requirements
### Requirement: Phase 1 Dashboard Tasks
The system SHALL treat the Phase 1 dashboard checklist in `SKILL.md` as an implementation scope for a minimal server-rendered dashboard, using Gin templates, HTMX partials, Tailwind styling, and the existing WebSocket feed instead of introducing a separate SPA frontend.

## REMOVED Requirements
None.
