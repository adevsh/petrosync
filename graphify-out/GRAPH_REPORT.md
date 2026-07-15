# Graph Report - .  (2026-07-14)

## Corpus Check
- 169 files · ~104,695 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1455 nodes · 3269 edges · 68 communities (64 shown, 4 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 229 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

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

## God Nodes (most connected - your core abstractions)
1. `fakeTankWorkflowQuerier` - 47 edges
2. `main()` - 35 edges
3. `New()` - 34 edges
4. `SetAuditAction()` - 28 edges
5. `SetAuditEntity()` - 28 edges
6. `SetAuditAfter()` - 27 edges
7. `DeliveryOrder` - 24 edges
8. `VehicleStatusT` - 21 edges
9. `Queries` - 20 edges
10. `ValkeyService` - 20 edges

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

## Communities (68 total, 4 thin omitted)

### Community 0 - "User Role Grants"
Cohesion: 0.06
Nodes (37): CheckUserHasCompanyRoleParams, CheckUserHasRoleInScopeParams, CreateUserParams, CreateUserRow, GetActiveRoleForUserAndScopeParams, GetUserByTelegramIDRow, GetUserRow, GrantRoleParams (+29 more)

### Community 1 - "Auth API"
Cohesion: 0.05
Nodes (29): AuthHandler, changePasswordRequest, loginRequest, loginResponse, logoutRequest, refreshRequest, Context, NewAuthHandler() (+21 more)

### Community 2 - "Trip Photos"
Cohesion: 0.06
Nodes (35): CreateTripPhotoParams, ListPhotosByTripAndCompartmentParams, ListPhotosByTripAndEventParams, PhotoEventT, TripPhoto, AllPhotoEventTValues(), Context, Queries (+27 more)

### Community 3 - "Weight Bridge Approvals"
Cohesion: 0.08
Nodes (33): ApprovalStatusT, ApproveWeightBridgeReadingParams, CreateWeightBridgeReadingParams, EscalateWeightBridgeReadingParams, ListEscalatedApprovalsRow, ListOverduePendingManualApprovalsRow, ListPendingManualApprovalsRow, MeasurementMethodT (+25 more)

### Community 4 - "Trip Lifecycle"
Cohesion: 0.10
Nodes (25): AppendTripRoutePointParams, CreateTripParams, DestinationTypeT, GetTripByDORow, GetTripRow, GetTripWithDetailsRow, ListActiveTripsByDriverUserScopeRow, ListActiveTripsByFacilityScopeRow (+17 more)

### Community 5 - "Vehicle Inventory"
Cohesion: 0.13
Nodes (29): CreateVehicleParams, GetVehicleByPlateRow, GetVehicleRow, ListDispatchCandidateVehiclesParams, ListDispatchCandidateVehiclesRow, ListVehiclesByDepotRow, ListVehiclesByStatusAndDepotParams, ListVehiclesByStatusAndDepotRow (+21 more)

### Community 6 - "Facility Topology"
Cohesion: 0.10
Nodes (24): CreateDepotParams, CreateFacilityParams, GetDepotByCodeRow, GetDepotRow, GetFacilityByCodeRow, GetFacilityRow, GetPrimaryFacilityByRefineryRow, ListAllActiveDepotsRow (+16 more)

### Community 7 - "GPS Event Partitions"
Cohesion: 0.16
Nodes (43): GpsEvent, GpsEvents202501, GpsEvents202502, GpsEvents202503, GpsEvents202504, GpsEvents202505, GpsEvents202506, GpsEvents202507 (+35 more)

### Community 8 - "Telegram Link Tokens"
Cohesion: 0.07
Nodes (25): CreateTelegramLinkTokenParams, GetValidTelegramLinkTokenRow, TelegramLinkToken, fakeTelegramLinkQuerier, fakeTelegramLinkQuerier, TelegramLinkQuerier, TelegramLinkTokenHandler, telegramLinkTokenResponse (+17 more)

### Community 9 - "Nullable DB Types"
Cohesion: 0.07
Nodes (14): NullApprovalStatusT, NullCompartmentDeliveryStatusT, NullDestinationTypeT, NullDocumentTypeT, NullDoStatusT, NullFuelCategoryT, NullNotificationTypeT, NullPhotoEventT (+6 more)

### Community 10 - "Audit Logging"
Cohesion: 0.09
Nodes (28): AsyncWriter, blockingSink, Sink, InsertAuditLogParams, InsertAuditLogRow, ListAuditLogByActionParams, ListAuditLogByActionRow, ListAuditLogByEntityParams (+20 more)

### Community 11 - "Trip Events"
Cohesion: 0.09
Nodes (22): GetLatestTripEventByTypeParams, InsertTripEventParams, ListTripEventsByTripAndTypeParams, TankWorkflowStore, TripEvent, TripEventTypeT, AllTripEventTypeTValues(), RawMessage (+14 more)

### Community 12 - "GPS Storage"
Cohesion: 0.10
Nodes (25): GetLatestGPSEventByTripRow, InsertGPSEventParams, InsertGPSEventRow, ListGPSEventsByTripAndTimeRangeParams, ListGPSEventsByTripAndTimeRangeRow, ListGPSEventsByTripRow, Float8, fakeGPSPublisher (+17 more)

### Community 13 - "Notification Logging"
Cohesion: 0.09
Nodes (22): CountNotificationsByTypeAndTripParams, InsertNotificationParams, ListNotificationsByRecipientParams, NotificationTypeT, AllNotificationTypeTValues(), Context, Queries, Int8 (+14 more)

### Community 14 - "Compartment Deliveries"
Cohesion: 0.10
Nodes (18): CompartmentDeliveryStatusT, CreateCompartmentDeliveryParams, GetCompartmentDeliveryByTripAndCompartmentParams, GetTripVarianceSummaryRow, ListCompartmentDeliveriesByTripRow, ListDisputedDeliveriesRow, NullMeasurementMethodT, UpdateCompartmentDeliveryStatusParams (+10 more)

### Community 15 - "User Admin APIs"
Cohesion: 0.14
Nodes (27): createUserRequest, NotifyReset, ResetPasswordHandler, resetPasswordRequest, roleChangeRequest, roleGrantResponse, updateUserRequest, UserCache (+19 more)

### Community 16 - "Refinery Fuel Setup"
Cohesion: 0.11
Nodes (16): CreateFuelTypeParams, CreateRefineryParams, FuelCategoryT, FuelType, Refinery, UpdateFuelTypeParams, UpdateRefineryParams, Context (+8 more)

### Community 17 - "Delivery Orders"
Cohesion: 0.15
Nodes (15): ApproveDeliveryOrderParams, AssignVehicleAndDriverToDOParams, CreateDeliveryOrderParams, DeliveryOrder, DoStatusT, ListDOsByStatusRow, ListDOsForDispatchQueueRow, UpdateDOStatusParams (+7 more)

### Community 18 - "Station Management"
Cohesion: 0.18
Nodes (17): CreateStationParams, GasStation, GetStationByCodeRow, GetStationRow, ListAllActiveStationsByRefineryScopeRow, ListAllActiveStationsByStationScopeRow, ListAllActiveStationsRow, ListStationsByFacilityRow (+9 more)

### Community 19 - "Station Tanks"
Cohesion: 0.17
Nodes (13): CreateStationTankParams, GetStationTankByFuelParams, ListStationTanksBelowReorderThresholdRow, StationTank, UpdateDipReadingParams, UpdateStationTankReorderThresholdParams, UpdateStationTankVolumeAfterDeliveryParams, UpdateStationTankVolumeParams (+5 more)

### Community 20 - "Dashboard Analytics"
Cohesion: 0.14
Nodes (18): GetCompanyWideDashboardSummaryRow, GetDriverComplianceSummaryParams, GetDriverComplianceSummaryRow, GetFacilityDashboardSummaryRow, GetMonthlyDeliveryStatsByFacilityParams, GetMonthlyDeliveryStatsByFacilityRow, GetStationInventorySnapshotRow, ListPendingWeightBridgeApprovalsByFacilityRow (+10 more)

### Community 21 - "Driver Operations"
Cohesion: 0.21
Nodes (14): CreateDriverParams, GetDriverByUserIDRow, GetDriverRow, ListAvailableDriversForDispatchRow, ListDriversByDepotRow, ListDriversWithExpiringLicenseRow, UpdateDriverHomeDepotParams, UpdateDriverLicenseParams (+6 more)

### Community 22 - "Mutating Handlers"
Cohesion: 0.21
Nodes (10): Context, floatToNumeric(), Numeric, Context, Context, SetAuditAction(), SetAuditAfter(), SetAuditBefore() (+2 more)

### Community 23 - "Runtime Stack Docs"
Cohesion: 0.14
Nodes (23): build-api trigger paths, Woodpecker pipeline, Docker development stack, Garage service, PostgreSQL PostGIS service, Valkey service, PetroSync README, planned stack (+15 more)

### Community 24 - "Storage Tanks"
Cohesion: 0.21
Nodes (12): CreateStorageTankParams, CreditStorageTankVolumeParams, DeductStorageTankVolumeParams, FacilityStorageTank, GetStorageTankAvailableVolumeRow, GetStorageTankByFacilityAndFuelParams, ReleaseStorageTankReservationParams, ReserveStorageTankVolumeParams (+4 more)

### Community 25 - "Workflow Tests"
Cohesion: 0.16
Nodes (7): GetMandatoryPhotoCheckByTripRow, ListTripDeliveredVolumeByFuelRow, ListTripLoadedVolumeByFuelRow, Context, Int8, fakeTankWorkflowQuerier, fakeTankWorkflowStore

### Community 26 - "Workflow Wiring"
Cohesion: 0.24
Nodes (22): Queries, New(), NewWorkflowService(), Numeric, T, numericInt64(), numericInt64Exp(), TestWorkflowService_ApproveDeliveryOrder_InsufficientStock() (+14 more)

### Community 27 - "Graphify References"
Cohesion: 0.14
Nodes (22): add-watch reference, watch mode, graph exports, exports reference, extraction spec reference, subagent extraction schema, cross-repo graph merge, github-and-merge reference (+14 more)

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
Cohesion: 0.20
Nodes (10): GetSealByTripAndCompartmentParams, IssueSealParams, ListSealsByTripRow, RecordSealBreakParams, VerifySealParams, Context, Queries, Int8 (+2 more)

### Community 33 - "Vehicle Compartments"
Cohesion: 0.21
Nodes (8): CreateCompartmentParams, GetTotalCapacityByVehicleRow, ListCompartmentsByVehicleAndFuelParams, UpdateCompartmentFuelTypeParams, Context, Queries, Numeric, Text

### Community 34 - "API Entry Auth"
Cohesion: 0.28
Nodes (13): main(), HandlerFunc, JWTAuth(), JWTQueryAuth(), T, makeToken(), TestJWTAuth_BlocksInactiveUsers(), TestJWTAuth_LoadsFromDBWhenCacheMiss() (+5 more)

### Community 35 - "Vehicle Maintenance"
Cohesion: 0.25
Nodes (9): CompleteMaintenanceRecordParams, CreateMaintenanceRecordParams, ListAllOpenMaintenanceRow, VehicleMaintenanceRecord, Context, Queries, Int8, Text (+1 more)

### Community 36 - "Route Deviations"
Cohesion: 0.20
Nodes (8): CreateDeviationEventParams, ListUnnotifiedDeviationsAboveThresholdRow, Context, Queries, Int4, Numeric, Text, Timestamptz

### Community 37 - "Valkey WS Bridge"
Cohesion: 0.15
Nodes (12): Completed, Context, RunValkeyBridge(), Completed, Context, T, TestRunValkeyBridge_BroadcastsMessages(), PubSubMessage (+4 more)

### Community 38 - "Telegram Link Tests"
Cohesion: 0.19
Nodes (9): fakeLinkQuerier, fakeLinkStore, fakeReplier, Context, Int8, T, TelegramLinkQuerier, TestHandleUpdate_LinkSuccess() (+1 more)

### Community 39 - "Core Models"
Cohesion: 0.19
Nodes (14): AuditLog, Driver, NotificationLog, RefineryFacility, RouteDeviationEvent, StationQrCode, User, Vehicle (+6 more)

### Community 40 - "Trip Mobile APIs"
Cohesion: 0.28
Nodes (4): Context, TripHandler, Queries, NewTripHandler()

### Community 41 - "RBAC Middleware"
Cohesion: 0.31
Nodes (11): DisallowDriver(), HandlerFunc, RequiredRole(), RoleRank(), T, TestDisallowDriver_AllowsMixedRoleUser(), TestDisallowDriver_BlocksDriverOnlyUser(), TestRequiredRole_HierarchyAndScopeMatch() (+3 more)

### Community 42 - "Telegram Bot Runtime"
Cohesion: 0.26
Nodes (7): Replier, TelegramBot, Context, HandleUpdate(), NewTelegramBot(), Context, Client

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
Cohesion: 0.31
Nodes (4): DeliveryOrderHandler, Context, Queries, NewDeliveryOrderHandler()

### Community 47 - "Vehicle Handler"
Cohesion: 0.31
Nodes (4): VehicleHandler, Context, Queries, NewVehicleHandler()

### Community 48 - "Driver Handler"
Cohesion: 0.38
Nodes (4): DriverHandler, Context, Queries, NewDriverHandler()

### Community 49 - "WebSocket Hub"
Cohesion: 0.25
Nodes (5): Conn, Context, NewHub(), RWMutex, Hub

### Community 50 - "Refinery Handler"
Cohesion: 0.36
Nodes (4): RefineryHandler, Context, Queries, NewRefineryHandler()

### Community 51 - "Seal Status Types"
Cohesion: 0.25
Nodes (4): CompartmentSeal, NullSealStatusT, SealStatusT, AllSealStatusTValues()

### Community 52 - "Transaction Store"
Cohesion: 0.32
Nodes (6): Store, TankWorkflowQuerier, Context, Pool, Queries, NewStore()

### Community 53 - "Station Handler"
Cohesion: 0.39
Nodes (4): StationHandler, Context, Queries, NewStationHandler()

### Community 54 - "Telegram Client"
Cohesion: 0.36
Nodes (7): User, Chat, getUpdatesResponse, Message, sendMessageResponse, Update, User

### Community 55 - "API Config Bootstrap"
Cohesion: 0.33
Nodes (5): main(), Config, Duration, Load(), NewClient()

### Community 56 - "QR Validation"
Cohesion: 0.38
Nodes (5): QRHandler, qrValidateReq, Context, Queries, NewQRHandler()

### Community 57 - "Storage Tank Handler"
Cohesion: 0.43
Nodes (4): StorageTankHandler, Context, Queries, NewStorageTankHandler()

### Community 58 - "SQLC DB Core"
Cohesion: 0.50
Nodes (3): DBTX, Queries, Tx

### Community 59 - "Region Lookup"
Cohesion: 0.60
Nodes (3): Region, Context, Queries

### Community 60 - "User Management Tests"
Cohesion: 0.60
Nodes (4): T, TestUserHandler_CreateUser(), TestUserHandler_GrantRole(), TestUserHandler_GrantRole_RequiresScopeIDForNonCompany()

### Community 62 - "Session Auth"
Cohesion: 0.67
Nodes (3): HandlerFunc, SessionAuth(), SessionStore

## Ambiguous Edges - Review These
- `schema.sql DDL source of truth` → `sql/migrations schema input`  [AMBIGUOUS]
  sqlc.yaml · relation: conceptually_related_to
- `build-api trigger paths` → `documented build-api trigger paths`  [AMBIGUOUS]
  .woodpecker.yml · relation: conceptually_related_to

## Knowledge Gaps
- **21 isolated node(s):** `garage-init.sh script`, `github.com/adevsh/petrosync`, `StationFacilityWhitelist`, `Querier`, `gpsEvent` (+16 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `schema.sql DDL source of truth` and `sql/migrations schema input`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `build-api trigger paths` and `documented build-api trigger paths`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `main()` connect `API Entry Auth` to `Auth API`, `Trip Photos`, `Telegram Link Tokens`, `Audit Logging`, `GPS Storage`, `Notification Logging`, `User Admin APIs`, `Workflow Wiring`, `Valkey WS Bridge`, `Trip Mobile APIs`, `RBAC Middleware`, `Telegram Bot Runtime`, `Telegram Link Store`, `Delivery Order Handler`, `Vehicle Handler`, `Driver Handler`, `WebSocket Hub`, `Refinery Handler`, `Transaction Store`, `Station Handler`, `API Config Bootstrap`, `QR Validation`, `Storage Tank Handler`?**
  _High betweenness centrality (0.148) - this node is a cross-community bridge._
- **Why does `UserRoleGrant` connect `User Role Grants` to `Auth API`, `GPS Event Partitions`, `User Admin APIs`, `Core Models`?**
  _High betweenness centrality (0.138) - this node is a cross-community bridge._
- **Why does `New()` connect `Workflow Wiring` to `Auth API`, `API Entry Auth`, `Facility Topology`, `Telegram Link Tokens`, `RBAC Middleware`, `Audit Logging`, `GPS Storage`, `Notification Logging`, `Transaction Store`, `API Config Bootstrap`, `SQLC DB Core`, `User Management Tests`?**
  _High betweenness centrality (0.094) - this node is a cross-community bridge._
- **Are the 34 inferred relationships involving `main()` (e.g. with `NewAsyncWriter()` and `NewTelegramBot()`) actually correct?**
  _`main()` has 34 INFERRED edges - model-reasoned connections that need verification._
- **What connects `garage-init.sh script`, `github.com/adevsh/petrosync`, `StationFacilityWhitelist` to the rest of the system?**
  _21 weakly-connected nodes found - possible documentation gaps or missing edges._