(function () {
  function parseSeed() {
    const node = document.getElementById("active-trips-map-seed");
    if (!node) {
      return [];
    }

    try {
      const payload = JSON.parse(node.textContent || "[]");
      return Array.isArray(payload) ? payload : [];
    } catch (_err) {
      return [];
    }
  }

  function buildWSURL(path) {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return protocol + "//" + window.location.host + path;
  }

  function formatSpeed(value) {
    if (typeof value !== "number" || Number.isNaN(value)) {
      return "-";
    }
    return value.toFixed(1) + " km/h";
  }

  function formatLastGPS(value) {
    if (!value) {
      return "Awaiting GPS fix";
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "Awaiting GPS fix";
    }

    return date.toLocaleString();
  }

  function hasCoordinates(trip) {
    return typeof trip.lat === "number" && typeof trip.lng === "number";
  }

  function popupHTML(trip) {
    const plate = escapeHTML(trip.plate_number || ("Trip #" + trip.trip_id));
    const driver = escapeHTML(trip.driver_name || "Driver pending");
    const destination = escapeHTML(trip.destination_name || "Destination pending");
    const status = escapeHTML(trip.status || "UNKNOWN");
    return [
      '<div class="text-sm">',
      "<strong>" + plate + "</strong><br>",
      driver + "<br>",
      destination + "<br>",
      "Status: " + status,
      "</div>",
    ].join("");
  }

  function escapeHTML(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function updateTripCard(trip) {
    const card = document.querySelector('[data-trip-id="' + trip.trip_id + '"]');
    if (!card) {
      return;
    }

    const speedNode = card.querySelector("[data-trip-speed]");
    if (speedNode) {
      speedNode.textContent = formatSpeed(trip.speed_kmh);
    }

    const lastGPSNode = card.querySelector("[data-trip-last-gps]");
    if (lastGPSNode) {
      lastGPSNode.textContent = formatLastGPS(trip.last_gps_at);
    }
  }

  function initMap() {
    const container = document.querySelector("[data-live-trip-map]");
    if (!container || container.dataset.mapReady === "true" || !window.L) {
      return;
    }

    const scope = container.dataset.liveMapScope || "company";
    const wsPath = container.dataset.liveMapWsPath || "/ws/trips/active";
    const statusNode = document.getElementById("live-map-status");
    const seed = parseSeed();
    const tripIndex = new Map();
    const markerIndex = new Map();
    let hasFitBounds = false;
    let reconnectTimer = null;

    function setStatus(text) {
      if (statusNode) {
        statusNode.textContent = text;
      }
    }

    const map = window.L.map(container, {
      zoomControl: true,
      scrollWheelZoom: false,
    }).setView([-2.5, 118], 5);

    window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
      attribution: "&copy; OpenStreetMap contributors",
    }).addTo(map);

    function maybeFitBounds() {
      if (hasFitBounds || markerIndex.size === 0) {
        return;
      }

      const bounds = window.L.latLngBounds(
        Array.from(markerIndex.values()).map(function (entry) {
          return entry.getLatLng();
        }),
      );
      map.fitBounds(bounds, { padding: [24, 24], maxZoom: 11 });
      hasFitBounds = true;
    }

    function upsertTrip(nextTrip) {
      const existing = tripIndex.get(nextTrip.trip_id) || {};
      const merged = Object.assign({}, existing, nextTrip);

      if (!tripIndex.has(nextTrip.trip_id) && scope === "facility" && merged.origin_facility_id == null) {
        return;
      }

      tripIndex.set(merged.trip_id, merged);
      updateTripCard(merged);

      if (!hasCoordinates(merged)) {
        return;
      }

      let marker = markerIndex.get(merged.trip_id);
      if (!marker) {
        marker = window.L.circleMarker([merged.lat, merged.lng], {
          radius: 7,
          weight: 2,
          color: "#0f172a",
          fillColor: "#2563eb",
          fillOpacity: 0.85,
        }).addTo(map);
        markerIndex.set(merged.trip_id, marker);
      } else {
        marker.setLatLng([merged.lat, merged.lng]);
      }

      marker.bindPopup(popupHTML(merged));
      maybeFitBounds();
    }

    seed.forEach(function (trip) {
      upsertTrip(trip);
    });

    if (markerIndex.size === 0) {
      setStatus("Waiting for GPS");
    }

    function connect() {
      setStatus("Connecting");
      const socket = new window.WebSocket(buildWSURL(wsPath));

      socket.addEventListener("open", function () {
        setStatus("Live");
      });

      socket.addEventListener("message", function (event) {
        try {
          const payload = JSON.parse(event.data);
          if (!payload || typeof payload.trip_id !== "number") {
            return;
          }

          const knownTrip = tripIndex.get(payload.trip_id);
          if (!knownTrip && scope === "facility") {
            return;
          }

          upsertTrip(
            Object.assign(
              {
                trip_id: payload.trip_id,
                plate_number: knownTrip && knownTrip.plate_number ? knownTrip.plate_number : "Trip #" + payload.trip_id,
                driver_name: knownTrip && knownTrip.driver_name ? knownTrip.driver_name : "",
                destination_name: knownTrip && knownTrip.destination_name ? knownTrip.destination_name : "",
                status: knownTrip && knownTrip.status ? knownTrip.status : "IN_TRANSIT",
                origin_facility_id: knownTrip ? knownTrip.origin_facility_id : null,
              },
              payload,
            ),
          );
        } catch (_err) {
          setStatus("Feed error");
        }
      });

      socket.addEventListener("error", function () {
        setStatus("Feed error");
      });

      socket.addEventListener("close", function () {
        setStatus("Reconnecting");
        reconnectTimer = window.setTimeout(connect, 2000);
      });

      container.__liveTripSocket = socket;
    }

    connect();
    container.dataset.mapReady = "true";
    container.__liveTripMap = map;

    window.addEventListener("beforeunload", function () {
      if (reconnectTimer) {
        window.clearTimeout(reconnectTimer);
      }
      if (container.__liveTripSocket) {
        container.__liveTripSocket.close();
      }
    });
  }

  document.addEventListener("DOMContentLoaded", initMap);
  document.body.addEventListener("htmx:load", initMap);
  document.body.addEventListener("htmx:beforeSwap", function (event) {
    const target = event.detail && event.detail.target;
    if (target && target.id === "active-trips-map") {
      event.detail.shouldSwap = false;
    }
  });
})();
