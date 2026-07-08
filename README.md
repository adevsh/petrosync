# PetroSync

PetroSync is a fuel distribution tracking system for multi-refinery petroleum operations, covering the full delivery flow from refinery loading to gas station delivery.

## What We Are Building

This repository is the backend and dashboard for PetroSync:

- Go API for core business logic and mobile integration
- HTMX dashboard for operations, dispatch, and monitoring
- Real-time trip and fleet tracking
- Delivery order, trip, vehicle, driver, and station management
- Audit-friendly workflows for petroleum logistics

## Planned Stack

Based on `SKILL.md`, PetroSync is planned with:

- Go + Gin for the API
- PostgreSQL + PostGIS for data
- sqlc for database access
- HTMX + Go templates for the dashboard
- Tailwind CSS for styling
- Valkey for sessions, caching, and pub/sub
- Garage for object storage
- Telegram for operational notifications

## Development Direction

The project will be developed in phases:

1. Core delivery workflow
2. Safety and monitoring features
3. Intelligence and automation
4. Enterprise and regulatory integrations

## Local AI Development Setup

This project will be developed with `PI + LM Studio` running locally on:

- Lenovo Legion Slim 7
- AMD Ryzen 9 5900HX
- 32 GB memory
- NVIDIA GeForce RTX 3060 Mobile

This setup is intended to support local, practical AI-assisted development while keeping iteration fast during design and implementation.
