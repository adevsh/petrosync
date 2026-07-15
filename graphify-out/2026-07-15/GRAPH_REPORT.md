# Graph Report - petrosync  (2026-07-15)

## Corpus Check
- 181 files · ~134,152 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2021 nodes · 4632 edges · 141 communities (101 shown, 40 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 355 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `d36b544d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- User Role Grants
- Auth API
- Trip Photos
- Weight Bridge Approvals
- Trip Lifecycle
- Vehicle Inventory
- Facility Topology
- GPS Event Partitions
- Telegram Link Tokens
- Nullable DB Types
- Audit Logging
- Trip Events
- GPS Storage
- Notification Logging
- Compartment Deliveries
- User Admin APIs
- Refinery Fuel Setup
- Delivery Orders
- Station Management
- Station Tanks
- Dashboard Analytics
- Driver Operations
- Mutating Handlers
- Runtime Stack Docs
- Storage Tanks
- Workflow Tests
- Workflow Wiring
- Graphify References
- System Settings
- DO Item Mapping
- Loading Bays
- Trip Documents
- Compartment Seals
- Vehicle Compartments
- API Entry Auth
- Vehicle Maintenance
- Route Deviations
- Valkey WS Bridge
- Telegram Link Tests
- Core Models
- Trip Mobile APIs
- RBAC Middleware
- Telegram Bot Runtime
- Station QR Codes
- Telegram Link Store
- Station Whitelist
- Delivery Order Handler
- Vehicle Handler
- Driver Handler
- WebSocket Hub
- Refinery Handler
- Seal Status Types
- Transaction Store
- Station Handler
- Telegram Client
- API Config Bootstrap
- QR Validation
- Storage Tank Handler
- SQLC DB Core
- Region Lookup
- User Management Tests
- Trip Photo Handler
- Session Auth
- SQLC Querier
- Garage Init Script
- Repo Module
- PetroSync — SKILL.md
- Context
- users.sql.go
- UserRoleGrant
- Context
- SetAuditEntity
- Context
- 10. Business Logic Rules by Domain
- GarageStorage
- package.json
- graphify reference: extra exports and benchmark
- auth_test.go
- 6. Mobile API Contract
- PetroSync
- graphify reference: query, path, explain
- 11. Dashboard Architecture
- 21. Code Commenting Standards
- 5. Go API Architecture
- 7. Authentication & Session Architecture
- sql/migrations schema input
- 23. Visual Design System
- 2. Locked Tech Stack
- 4. Database Layer Rules
- 8. RBAC Enforcement Rules
- graphify reference: add a URL and watch a folder
- graphify reference: commit hook and native AGENTS.md integration
- graphify reference: incremental update and cluster-only
- build-api trigger paths
- 14. Valkey Architecture
- 17. Phase 1 — Core Loop
- 9. Canonical State Machines
- graphify reference: GitHub clone and cross-repo merge
- graphify reference: transcribe video and audio
- tasks.md
- AGENTS.md
- extraction-spec.md
- watch mode
- graph exports
- exports reference
- extraction spec reference
- subagent extraction schema
- cross-repo graph merge
- github-and-merge reference
- AGENTS.md integration
- hooks reference
- post-commit graph rebuild hook
- query reference
- graph vocabulary expansion
- transcribe reference
- Whisper transcription
- incremental update
- update reference
- AST extraction
- existing graph fast path
- graphify skill
- semantic extraction
- AGENTS graphify rules
- PetroSync README
- Android JWT authentication
- background worker cron jobs
- dashboard session authentication
- PetroSync SKILL reference
- real-time trip map
- Woodpecker path triggers
- map.js
- GarageStorage
- CompartmentDeliveryStatusT
- .loadTripForAccess
- loadSession
- QRHandler
- StorageTankHandler
- .UserDetail

## God Nodes (most connected - your core abstractions)
1. `New()` - 66 edges
2. `fakeDashboardDataQuerier` - 51 edges
3. `fakeTankWorkflowQuerier` - 47 edges
4. `dashboardPageData` - 44 edges
5. `fakeDashboardWorkflowQuerier` - 41 edges
6. `main()` - 38 edges
7. `DeliveryOrder` - 35 edges
8. `SessionData` - 31 edges
9. `DashboardHandler` - 30 edges
10. `SetAuditAction()` - 28 edges

## Surprising Connections (you probably didn't know these)
- `Valkey service` --semantically_similar_to--> `Valkey session and pub-sub architecture`  [INFERRED] [semantically similar]
  docker-compose.yml → SKILL.md
- `Garage service` --semantically_similar_to--> `Garage object storage`  [INFERRED] [semantically similar]
  docker-compose.yml → SKILL.md
- `PostgreSQL PostGIS service` --semantically_similar_to--> `PostgreSQL with PostGIS`  [INFERRED] [semantically similar]
  docker-compose.yml → SKILL.md
- `main()` --calls--> `NewAsyncWriter()`  [INFERRED]
  cmd/api/main.go → internal/auditlog/async_writer.go
- `main()` --calls--> `NewTelegramBot()`  [INFERRED]
  cmd/api/main.go → internal/bot/telegram.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **graphify reference modules** — _trae_skills_graphify_references_add_watch_add_watch_reference, _trae_skills_graphify_references_exports_exports_reference, _trae_skills_graphify_references_extraction_spec_extraction_spec_reference, _trae_skills_graphify_references_github_and_merge_github_and_merge_reference, _trae_skills_graphify_references_hooks_hooks_reference, _trae_skills_graphify_references_query_query_reference, _trae_skills_graphify_references_transcribe_transcribe_reference, _trae_skills_graphify_references_update_update_reference [EXTRACTED 1.00]
- **Docker development services** — docker_compose_postgres_service, docker_compose_valkey_service, docker_compose_garage_service [EXTRACTED 1.00]
- **PetroSync runtime stack** — skill_postgresql_postgis, skill_sqlc_only_data_access, skill_valkey_architecture, skill_garage_object_storage, skill_realtime_trip_map, skill_background_worker_cron [INFERRED 0.85]

## Communities (141 total, 40 thin omitted)

### Community 0 - "User Role Grants"
Cohesion: 0.07
Nodes (40): dashboardDeliveryOrderDetailView, dashboardDeliveryOrderItemView, dashboardDeliveryOrderRow, dashboardFleetAttentionView, dashboardFleetMaintenanceView, dashboardFleetVehicleRow, dashboardNotificationService, dashboardPageData (+32 more)

### Community 1 - "Auth API"
Cohesion: 0.20
Nodes (4): Context, Duration, NewValkeyService(), ValkeyService

### Community 2 - "Trip Photos"
Cohesion: 0.06
Nodes (36): CreateTripPhotoParams, ListPhotosByTripAndCompartmentParams, ListPhotosByTripAndEventParams, PhotoEventT, TripPhoto, fakeDashboardTripPhotoLister, AllPhotoEventTValues(), Context (+28 more)

### Community 3 - "Weight Bridge Approvals"
Cohesion: 0.12
Nodes (20): ApprovalStatusT, ApproveWeightBridgeReadingParams, CreateWeightBridgeReadingParams, EscalateWeightBridgeReadingParams, ListEscalatedApprovalsRow, ListOverduePendingManualApprovalsRow, ListPendingManualApprovalsRow, MeasurementMethodT (+12 more)

### Community 4 - "Trip Lifecycle"
Cohesion: 0.16
Nodes (6): AppendTripRoutePointParams, ListTripsByDriverParams, ListTripsByVehicleParams, Trip, Context, Queries

### Community 5 - "Vehicle Inventory"
Cohesion: 0.21
Nodes (24): CreateVehicleParams, GetVehicleByPlateRow, GetVehicleRow, ListDispatchCandidateVehiclesRow, ListVehiclesByDepotRow, ListVehiclesByStatusAndDepotParams, ListVehiclesByStatusAndDepotRow, ListVehiclesByStatusAndFacilityParams (+16 more)

### Community 6 - "Facility Topology"
Cohesion: 0.10
Nodes (24): CreateDepotParams, CreateFacilityParams, GetDepotByCodeRow, GetDepotRow, GetFacilityByCodeRow, GetFacilityRow, GetPrimaryFacilityByRefineryRow, ListAllActiveDepotsRow (+16 more)

### Community 7 - "GPS Event Partitions"
Cohesion: 0.14
Nodes (48): GpsEvent, GpsEvents202501, GpsEvents202502, GpsEvents202503, GpsEvents202504, GpsEvents202505, GpsEvents202506, GpsEvents202507 (+40 more)

### Community 8 - "Telegram Link Tokens"
Cohesion: 0.13
Nodes (12): CreateTelegramLinkTokenParams, fakeTelegramLinkQuerier, TelegramLinkQuerier, TelegramLinkTokenHandler, telegramLinkTokenResponse, Context, Queries, Time (+4 more)

### Community 9 - "Nullable DB Types"
Cohesion: 0.07
Nodes (14): NullApprovalStatusT, NullCompartmentDeliveryStatusT, NullDestinationTypeT, NullDocumentTypeT, NullDoStatusT, NullFuelCategoryT, NullNotificationTypeT, NullPhotoEventT (+6 more)

### Community 10 - "Audit Logging"
Cohesion: 0.09
Nodes (28): AsyncWriter, blockingSink, Sink, InsertAuditLogParams, InsertAuditLogRow, ListAuditLogByActionParams, ListAuditLogByActionRow, ListAuditLogByEntityParams (+20 more)

### Community 11 - "Trip Events"
Cohesion: 0.07
Nodes (28): GetLatestTripEventByTypeParams, InsertTripEventParams, ListTripEventsByTripAndTypeParams, Store, TankWorkflowQuerier, TankWorkflowStore, TripEvent, TripEventTypeT (+20 more)

### Community 12 - "GPS Storage"
Cohesion: 0.10
Nodes (25): GetLatestGPSEventByTripRow, InsertGPSEventParams, InsertGPSEventRow, ListGPSEventsByTripAndTimeRangeParams, ListGPSEventsByTripAndTimeRangeRow, ListGPSEventsByTripRow, Float8, fakeGPSPublisher (+17 more)

### Community 13 - "Notification Logging"
Cohesion: 0.06
Nodes (48): main(), displayPGInt8(), Context, Int8, ignoreNotificationSendError(), int64Ptr(), NewNotificationCoordinator(), Context (+40 more)

### Community 14 - "Compartment Deliveries"
Cohesion: 0.10
Nodes (20): CompartmentDeliveryStatusT, CreateCompartmentDeliveryParams, GetCompartmentDeliveryByTripAndCompartmentParams, GetTripVarianceSummaryRow, ListCompartmentDeliveriesByTripRow, ListDisputedDeliveriesRow, ListTripDeliveredVolumeByFuelRow, ListTripLoadedVolumeByFuelRow (+12 more)

### Community 15 - "User Admin APIs"
Cohesion: 0.14
Nodes (27): createUserRequest, NotifyReset, ResetPasswordHandler, resetPasswordRequest, roleChangeRequest, roleGrantResponse, updateUserRequest, UserCache (+19 more)

### Community 16 - "Refinery Fuel Setup"
Cohesion: 0.19
Nodes (9): CreateFuelTypeParams, FuelCategoryT, FuelType, UpdateFuelTypeParams, Context, Queries, Int2, Numeric (+1 more)

### Community 17 - "Delivery Orders"
Cohesion: 0.12
Nodes (18): ApproveDeliveryOrderParams, AssignVehicleAndDriverToDOParams, CreateDeliveryOrderParams, DeliveryOrder, DestinationTypeT, DoStatusT, ListDOsByStatusRow, ListDOsForDispatchQueueRow (+10 more)

### Community 18 - "Station Management"
Cohesion: 0.20
Nodes (15): CreateStationParams, GetStationByCodeRow, GetStationRow, ListAllActiveStationsByRefineryScopeRow, ListAllActiveStationsByStationScopeRow, ListAllActiveStationsRow, ListStationsByFacilityRow, ListStationsByRegionRow (+7 more)

### Community 19 - "Station Tanks"
Cohesion: 0.17
Nodes (13): CreateStationTankParams, GetStationTankByFuelParams, ListStationTanksBelowReorderThresholdRow, StationTank, UpdateDipReadingParams, UpdateStationTankReorderThresholdParams, UpdateStationTankVolumeAfterDeliveryParams, UpdateStationTankVolumeParams (+5 more)

### Community 20 - "Dashboard Analytics"
Cohesion: 0.21
Nodes (6): dashboardLookupCache, canViewFacility(), Context, DashboardHandler, newDashboardLookupCache(), tripDestinationLabel()

### Community 21 - "Driver Operations"
Cohesion: 0.21
Nodes (14): CreateDriverParams, GetDriverByUserIDRow, GetDriverRow, ListAvailableDriversForDispatchRow, ListDriversByDepotRow, ListDriversWithExpiringLicenseRow, UpdateDriverHomeDepotParams, UpdateDriverLicenseParams (+6 more)

### Community 22 - "Mutating Handlers"
Cohesion: 0.17
Nodes (12): DeliveryOrderHandler, Context, Queries, NewDeliveryOrderHandler(), Context, Context, Context, SetAuditAction() (+4 more)

### Community 23 - "Runtime Stack Docs"
Cohesion: 0.29
Nodes (7): Docker development stack, Garage service, PostgreSQL PostGIS service, Valkey service, Garage object storage, PostgreSQL with PostGIS, Valkey session and pub-sub architecture

### Community 24 - "Storage Tanks"
Cohesion: 0.19
Nodes (11): CreateStorageTankParams, CreditStorageTankVolumeParams, DeductStorageTankVolumeParams, GetStorageTankAvailableVolumeRow, GetStorageTankByFacilityAndFuelParams, ReleaseStorageTankReservationParams, ReserveStorageTankVolumeParams, UpdateStorageTankVolumeParams (+3 more)

### Community 25 - "Workflow Tests"
Cohesion: 0.17
Nodes (6): FacilityStorageTank, GetMandatoryPhotoCheckByTripRow, Context, Int8, fakeTankWorkflowQuerier, fakeTankWorkflowStore

### Community 26 - "Workflow Wiring"
Cohesion: 0.24
Nodes (20): NewWorkflowService(), Numeric, T, numericInt64(), numericInt64Exp(), TestWorkflowService_ApproveDeliveryOrder_InsufficientStock(), TestWorkflowService_ApproveDeliveryOrder_ReservesPerItem(), TestWorkflowService_CancelDeliveryOrder_ReleasesOnlyWhenReserved() (+12 more)

### Community 28 - "System Settings"
Cohesion: 0.22
Nodes (10): DeleteFacilitySettingParams, GetEffectiveSettingParams, GetFacilitySettingParams, SystemSetting, UpsertFacilitySettingParams, UpsertGlobalSettingParams, Context, Queries (+2 more)

### Community 29 - "DO Item Mapping"
Cohesion: 0.20
Nodes (10): AssignCompartmentToDOItemParams, CreateDeliveryOrderItemParams, DeliveryOrderItem, ListDOItemsByDORow, UpdateDOItemAllocatedVolumeParams, Context, Queries, Int8 (+2 more)

### Community 30 - "Loading Bays"
Cohesion: 0.22
Nodes (10): CreateLoadingBayParams, FacilityLoadingBay, GetLoadingBayByQRPayloadRow, UpdateLoadingBayFuelTypeParams, ValidateLoadingBayQRParams, ValidateLoadingBayQRRow, Context, Queries (+2 more)

### Community 31 - "Trip Documents"
Cohesion: 0.19
Nodes (10): CreateTripDocumentParams, DocumentTypeT, GetTripDocumentByTypeParams, TripDocument, UpdateTripDocumentKeyParams, AllDocumentTypeTValues(), Context, Queries (+2 more)

### Community 32 - "Compartment Seals"
Cohesion: 0.12
Nodes (27): dashboardActiveTrip, dashboardAuthService, DashboardBreadcrumb, dashboardCompanyFacilitySummary, dashboardDataQuerier, dashboardFacilitySummary, dashboardLiveMap, dashboardMetric (+19 more)

### Community 33 - "Vehicle Compartments"
Cohesion: 0.21
Nodes (8): CreateCompartmentParams, GetTotalCapacityByVehicleRow, ListCompartmentsByVehicleAndFuelParams, UpdateCompartmentFuelTypeParams, Context, Queries, Numeric, Text

### Community 34 - "API Entry Auth"
Cohesion: 0.25
Nodes (5): Conn, Context, NewHub(), RWMutex, Hub

### Community 35 - "Vehicle Maintenance"
Cohesion: 0.24
Nodes (8): CompleteMaintenanceRecordParams, CreateMaintenanceRecordParams, ListAllOpenMaintenanceRow, Context, Queries, Int8, Text, Timestamptz

### Community 36 - "Route Deviations"
Cohesion: 0.09
Nodes (25): CreateDeviationEventParams, ListActiveTripsOffRouteRow, ListUnnotifiedDeviationsAboveThresholdRow, RouteDeviationEvent, Int4, Context, Queries, Decimal (+17 more)

### Community 37 - "Valkey WS Bridge"
Cohesion: 0.32
Nodes (7): Completed, Context, RunValkeyBridge(), T, TestRunValkeyBridge_BroadcastsMessages(), PubSubReceiver, TextBroadcaster

### Community 38 - "Telegram Link Tests"
Cohesion: 0.19
Nodes (9): fakeLinkQuerier, fakeLinkStore, fakeReplier, Context, Int8, T, TelegramLinkQuerier, TestHandleUpdate_LinkSuccess() (+1 more)

### Community 39 - "Core Models"
Cohesion: 0.14
Nodes (16): AuditLog, Driver, GasStation, NotificationLog, NotificationTypeT, StationQrCode, User, Vehicle (+8 more)

### Community 40 - "Trip Mobile APIs"
Cohesion: 0.28
Nodes (4): Context, TripHandler, Queries, NewTripHandler()

### Community 41 - "RBAC Middleware"
Cohesion: 0.31
Nodes (11): DisallowDriver(), HandlerFunc, RequiredRole(), RoleRank(), T, TestDisallowDriver_AllowsMixedRoleUser(), TestDisallowDriver_BlocksDriverOnlyUser(), TestRequiredRole_HierarchyAndScopeMatch() (+3 more)

### Community 42 - "Telegram Bot Runtime"
Cohesion: 0.48
Nodes (5): Replier, TelegramBot, Context, HandleUpdate(), NewTelegramBot()

### Community 43 - "Station QR Codes"
Cohesion: 0.27
Nodes (5): CreateStationQRCodeParams, GetStationByQRPayloadRow, Context, Queries, Text

### Community 44 - "Telegram Link Store"
Cohesion: 0.30
Nodes (9): Context, Pool, Queries, NewPgxTelegramLinkStore(), NewTelegramLinkService(), PgxTelegramLinkStore, TelegramLinkQuerier, TelegramLinkService (+1 more)

### Community 45 - "Station Whitelist"
Cohesion: 0.29
Nodes (6): AddFacilityToStationWhitelistParams, CheckFacilityCanServeStationParams, ListFacilitiesForStationRow, RemoveFacilityFromStationWhitelistParams, Context, Queries

### Community 46 - "Delivery Order Handler"
Cohesion: 0.07
Nodes (28): ADDED Requirements, Impact, MODIFIED Requirements, Phase 1 Dashboard Spec, REMOVED Requirements, Requirement: Base Dashboard Layout, Requirement: Dashboard Asset Pipeline, Requirement: Delivery Order Workflow UI (+20 more)

### Community 47 - "Vehicle Handler"
Cohesion: 0.31
Nodes (4): VehicleHandler, Context, Queries, NewVehicleHandler()

### Community 48 - "Driver Handler"
Cohesion: 0.42
Nodes (4): DriverHandler, Context, Queries, NewDriverHandler()

### Community 49 - "WebSocket Hub"
Cohesion: 0.15
Nodes (13): CountNotificationsByTypeAndTripParams, InsertNotificationParams, ListNotificationsByRecipientParams, Context, Queries, Int8, Text, Context (+5 more)

### Community 50 - "Refinery Handler"
Cohesion: 0.15
Nodes (12): GetValidTelegramLinkTokenRow, fakeTelegramLinkQuerier, Int8, Timestamptz, Context, Int8, T, TelegramLinkQuerier (+4 more)

### Community 52 - "Transaction Store"
Cohesion: 0.06
Nodes (33): CheckUserHasCompanyRoleParams, CheckUserHasRoleInScopeParams, GetActiveRoleForUserAndScopeParams, GrantRoleParams, ListUsersWithCompanyRoleRow, ListUsersWithRoleInScopeParams, ListUsersWithRoleInScopeRow, RevokeRoleParams (+25 more)

### Community 53 - "Station Handler"
Cohesion: 0.22
Nodes (7): StationHandler, floatToNumeric(), Numeric, numericToFloat64(), Context, Queries, NewStationHandler()

### Community 54 - "Telegram Client"
Cohesion: 0.21
Nodes (10): Context, User, NewClient(), Chat, Client, getUpdatesResponse, Message, sendMessageResponse (+2 more)

### Community 55 - "API Config Bootstrap"
Cohesion: 0.18
Nodes (14): dashboardStationRow, fleetAttentionDate(), formatTripMoment(), Context, DashboardHandler, dashboardBestRole(), formatDate(), formatNumeric() (+6 more)

### Community 56 - "QR Validation"
Cohesion: 0.16
Nodes (14): GetDriverComplianceSummaryParams, GetDriverComplianceSummaryRow, GetMonthlyDeliveryStatsByFacilityParams, GetMonthlyDeliveryStatsByFacilityRow, GetStationInventorySnapshotRow, ListPendingWeightBridgeApprovalsByFacilityRow, Context, Queries (+6 more)

### Community 57 - "Storage Tank Handler"
Cohesion: 0.50
Nodes (3): DBTX, Queries, Tx

### Community 58 - "SQLC DB Core"
Cohesion: 0.08
Nodes (24): For /graphify add and --watch, For /graphify query, For the commit hook and native AGENTS.md integration, For --update and --cluster-only, /graphify, Honesty Rules, Interpreter guard for subcommands, Part A - Structural extraction for code files (+16 more)

### Community 59 - "Region Lookup"
Cohesion: 0.60
Nodes (3): Region, Context, Queries

### Community 60 - "User Management Tests"
Cohesion: 0.09
Nodes (12): GetCompanyWideDashboardSummaryRow, GetFacilityDashboardSummaryRow, ListSealsByTripRow, ListVehiclesWithMaintenanceOrExpiryDueRow, fakeDashboardDataQuerier, fakeDashboardWorkflowQuerier, Timestamptz, Queries (+4 more)

### Community 62 - "Session Auth"
Cohesion: 0.16
Nodes (32): fakeDashboardAuthService, fakeDashboardSessionStore, Decimal, Duration, Engine, Numeric, T, mustDecimalFromString() (+24 more)

### Community 68 - "PetroSync — SKILL.md"
Cohesion: 0.11
Nodes (17): 12. Telegram Bot Rules, 13. Object Storage Rules — Garage, 15. Real-time WebSocket Architecture, 16. Background Worker Architecture, 18. Phase 2 — Safety Layer, 19. Phase 3 — Intelligence, 1. Project Identity, 20. Phase 4 — Enterprise (+9 more)

### Community 69 - "Context"
Cohesion: 0.18
Nodes (10): CompartmentSeal, GetSealByTripAndCompartmentParams, IssueSealParams, NullSealStatusT, RecordSealBreakParams, VerifySealParams, Context, Queries (+2 more)

### Community 70 - "users.sql.go"
Cohesion: 0.16
Nodes (15): dashboardNavItem, canManageUsers(), canViewFleet(), canViewStation(), canViewStationPages(), dashboardNav(), HandlerFunc, canViewDeliveryOrderPages() (+7 more)

### Community 71 - "UserRoleGrant"
Cohesion: 0.36
Nodes (4): RefineryHandler, Context, Queries, NewRefineryHandler()

### Community 72 - "Context"
Cohesion: 0.07
Nodes (29): CreateUserParams, CreateUserRow, GetUserByTelegramIDRow, GetUserRow, LinkTelegramAccountParams, ListActiveUsersRow, ListUsersRow, SetUserActiveParams (+21 more)

### Community 73 - "SetAuditEntity"
Cohesion: 0.18
Nodes (4): ListDispatchCandidateVehiclesParams, UpdateVehicleLocationParams, Context, Queries

### Community 74 - "Context"
Cohesion: 0.18
Nodes (23): HandlerFunc, JWTAuth(), JWTQueryAuth(), SessionOrJWTQueryAuth(), T, makeToken(), TestJWTAuth_BlocksInactiveUsers(), TestJWTAuth_LoadsFromDBWhenCacheMiss() (+15 more)

### Community 75 - "10. Business Logic Rules by Domain"
Cohesion: 0.20
Nodes (10): 10. Business Logic Rules by Domain, Audit Log, Dispatch Candidate Selection, Mandatory Photo Enforcement, QR Code Validation, Route Deviation Policy, Seal Tracking, Storage Tank Reservation (+2 more)

### Community 76 - "GarageStorage"
Cohesion: 0.21
Nodes (8): AuthHandler, changePasswordRequest, loginRequest, loginResponse, logoutRequest, refreshRequest, Context, NewAuthHandler()

### Community 77 - "package.json"
Cohesion: 0.22
Nodes (8): devDependencies, tailwindcss, @tailwindcss/cli, name, packageManager, private, tailwindcss, @tailwindcss/cli

### Community 78 - "graphify reference: extra exports and benchmark"
Cohesion: 0.22
Nodes (8): graphify reference: extra exports and benchmark, Step 6b - Wiki (only if --wiki flag), Step 7 - Neo4j export (only if --neo4j or --neo4j-push flag), Step 7a - FalkorDB export (only if --falkordb or --falkordb-push flag), Step 7b - SVG export (only if --svg flag), Step 7c - GraphML export (only if --graphml flag), Step 7d - MCP server (only if --mcp flag), Step 8 - Token reduction benchmark (only if total_words > 5000)

### Community 79 - "auth_test.go"
Cohesion: 0.29
Nodes (16): CreateTripParams, GetTripByDORow, GetTripRow, GetTripWithDetailsRow, ListActiveTripsByDriverUserScopeRow, ListActiveTripsByFacilityScopeRow, ListActiveTripsByRefineryScopeRow, ListActiveTripsByStationScopeRow (+8 more)

### Community 80 - "6. Mobile API Contract"
Cohesion: 0.29
Nodes (7): 6. Mobile API Contract, Active Trip Polling, Authentication Header, GPS Batch Endpoint, Photo Upload Endpoint, QR Validation, Trip Event Endpoint

### Community 81 - "PetroSync"
Cohesion: 0.33
Nodes (5): Development Direction, Local AI Development Setup, PetroSync, Planned Stack, What We Are Building

### Community 82 - "graphify reference: query, path, explain"
Cohesion: 0.33
Nodes (5): For /graphify explain, For /graphify path, graphify reference: query, path, explain, Step 0 — Constrained query expansion (REQUIRED before traversal), Step 1 — Traversal

### Community 83 - "11. Dashboard Architecture"
Cohesion: 0.40
Nodes (5): 11. Dashboard Architecture, HTMX Patterns, Map (JS Island), Page Map, Template Rules

### Community 84 - "21. Code Commenting Standards"
Cohesion: 0.40
Nodes (5): 21. Code Commenting Standards, File-Level Doc Block, Function-Level Doc, Inline Comments — Why, Not What, Test Functions

### Community 85 - "5. Go API Architecture"
Cohesion: 0.40
Nodes (5): 5. Go API Architecture, Error Codes, Handler Rules, Standard Response Envelope, URL Design

### Community 86 - "7. Authentication & Session Architecture"
Cohesion: 0.40
Nodes (5): 7. Authentication & Session Architecture, Android (JWT), Dashboard (HTMX), Password Reset Flow, Telegram Bot Linking

### Community 87 - "sql/migrations schema input"
Cohesion: 0.40
Nodes (5): schema.sql DDL source of truth, sqlc-only data access, sql/migrations schema input, sqlc configuration, sqlc type overrides

### Community 88 - "23. Visual Design System"
Cohesion: 0.50
Nodes (4): 23. Visual Design System, Colour Palette, Component Rules, Status Colours

### Community 89 - "2. Locked Tech Stack"
Cohesion: 0.50
Nodes (4): 2. Locked Tech Stack, Backend API, Dashboard, Infrastructure

### Community 90 - "4. Database Layer Rules"
Cohesion: 0.50
Nodes (4): 4. Database Layer Rules, Absolute Rules, Connection Configuration, Transaction Rules

### Community 91 - "8. RBAC Enforcement Rules"
Cohesion: 0.50
Nodes (4): 8. RBAC Enforcement Rules, Enforcement Rules, Middleware Chain, Role Hierarchy

### Community 92 - "graphify reference: add a URL and watch a folder"
Cohesion: 0.50
Nodes (3): For /graphify add, For --watch, graphify reference: add a URL and watch a folder

### Community 93 - "graphify reference: commit hook and native AGENTS.md integration"
Cohesion: 0.50
Nodes (3): For git commit hook, For native AGENTS.md integration (Trae), graphify reference: commit hook and native AGENTS.md integration

### Community 94 - "graphify reference: incremental update and cluster-only"
Cohesion: 0.50
Nodes (3): For --cluster-only, For --update (incremental re-extraction), graphify reference: incremental update and cluster-only

### Community 95 - "build-api trigger paths"
Cohesion: 0.67
Nodes (3): build-api trigger paths, Woodpecker pipeline, documented build-api trigger paths

### Community 96 - "14. Valkey Architecture"
Cohesion: 0.67
Nodes (3): 14. Valkey Architecture, Key Namespaces, Pub/Sub for Real-time Map

### Community 97 - "17. Phase 1 — Core Loop"
Cohesion: 0.67
Nodes (3): 17. Phase 1 — Core Loop, API Tasks, Dashboard Tasks

### Community 98 - "9. Canonical State Machines"
Cohesion: 0.67
Nodes (3): 9. Canonical State Machines, Delivery Order, Trip

### Community 132 - "map.js"
Cohesion: 0.29
Nodes (7): escapeHTML(), formatLastGPS(), formatSpeed(), initMap(), parseSeed(), popupHTML(), updateTripCard()

### Community 134 - "GarageStorage"
Cohesion: 0.31
Nodes (6): CreateRefineryParams, Refinery, UpdateRefineryParams, Context, Queries, Int2

### Community 135 - "CompartmentDeliveryStatusT"
Cohesion: 0.29
Nodes (5): Completed, Context, PubSubMessage, fakePubSubReceiver, fakeTextBroadcaster

### Community 136 - ".loadTripForAccess"
Cohesion: 0.40
Nodes (4): main(), Config, Duration, Load()

### Community 137 - "loadSession"
Cohesion: 0.40
Nodes (5): dashboardFleetVehicleSummary, Date, Int8, Numeric, Text

### Community 138 - "QRHandler"
Cohesion: 0.38
Nodes (5): QRHandler, qrValidateReq, Context, Queries, NewQRHandler()

### Community 139 - "StorageTankHandler"
Cohesion: 0.43
Nodes (4): StorageTankHandler, Context, Queries, NewStorageTankHandler()

## Ambiguous Edges - Review These
- `schema.sql DDL source of truth` → `sql/migrations schema input`  [AMBIGUOUS]
  sqlc.yaml · relation: conceptually_related_to
- `build-api trigger paths` → `documented build-api trigger paths`  [AMBIGUOUS]
  .woodpecker.yml · relation: conceptually_related_to

## Knowledge Gaps
- **182 isolated node(s):** `garage-init.sh script`, `github.com/adevsh/petrosync`, `StationFacilityWhitelist`, `Querier`, `gpsEvent` (+177 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **40 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `schema.sql DDL source of truth` and `sql/migrations schema input`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `build-api trigger paths` and `documented build-api trigger paths`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `User Management Tests` to `User Role Grants`, `Facility Topology`, `.loadTripForAccess`, `Context`, `Audit Logging`, `Trip Events`, `GPS Storage`, `Notification Logging`, `Telegram Link Tokens`, `Context`, `RBAC Middleware`, `WebSocket Hub`, `Dashboard Analytics`, `Transaction Store`, `API Config Bootstrap`, `Storage Tank Handler`, `Workflow Wiring`, `Session Auth`?**
  _High betweenness centrality (0.183) - this node is a cross-community bridge._
- **Why does `main()` connect `.loadTripForAccess` to `Auth API`, `Trip Photos`, `Telegram Link Tokens`, `Audit Logging`, `Trip Events`, `GPS Storage`, `QRHandler`, `StorageTankHandler`, `User Admin APIs`, `Notification Logging`, `Mutating Handlers`, `Workflow Wiring`, `Compartment Seals`, `API Entry Auth`, `Valkey WS Bridge`, `Trip Mobile APIs`, `RBAC Middleware`, `Telegram Bot Runtime`, `Telegram Link Store`, `Vehicle Handler`, `Driver Handler`, `Transaction Store`, `Station Handler`, `Telegram Client`, `User Management Tests`, `UserRoleGrant`, `Context`, `GarageStorage`?**
  _High betweenness centrality (0.140) - this node is a cross-community bridge._
- **Why does `fakeDashboardDataQuerier` connect `User Management Tests` to `Vehicle Maintenance`, `Trip Lifecycle`, `Vehicle Inventory`, `Facility Topology`, `GPS Event Partitions`, `Core Models`, `auth_test.go`, `Station Management`, `Station Tanks`, `QR Validation`, `Session Auth`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Are the 63 inferred relationships involving `New()` (e.g. with `main()` and `main()`) actually correct?**
  _`New()` has 63 INFERRED edges - model-reasoned connections that need verification._
- **What connects `garage-init.sh script`, `github.com/adevsh/petrosync`, `StationFacilityWhitelist` to the rest of the system?**
  _182 weakly-connected nodes found - possible documentation gaps or missing edges._