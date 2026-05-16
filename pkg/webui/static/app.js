(() => {
  const metrics = [
    {
      id: "memory",
      title: "Memory",
      subtitle: "Used GPU memory",
      unit: "GiB",
      domain: "auto",
      value(device) {
        return device.memory ? bytesToGiB(device.memory.used_bytes) : null;
      },
      detail(device) {
        if (!device.memory) return "n/a";
        return `${bytesToGiB(device.memory.used_bytes).toFixed(2)} / ${bytesToGiB(device.memory.total_bytes).toFixed(2)} GiB`;
      },
    },
    {
      id: "gpu-util",
      title: "GPU Util",
      subtitle: "SM activity",
      unit: "%",
      domain: [0, 100],
      value(device) {
        return numberOrNull(device.utilization && device.utilization.gpu_percent);
      },
    },
    {
      id: "mem-util",
      title: "MEM Util",
      subtitle: "Memory controller activity",
      unit: "%",
      domain: [0, 100],
      value(device) {
        return numberOrNull(device.utilization && device.utilization.memory_percent);
      },
    },
    {
      id: "temp",
      title: "Temperature",
      subtitle: "GPU core temperature",
      unit: "C",
      domain: "auto",
      value(device) {
        return numberOrNull(device.temperature && device.temperature.gpu_celsius);
      },
    },
    {
      id: "power",
      title: "Power",
      subtitle: "Current board draw",
      unit: "W",
      domain: "auto",
      value(device) {
        const mw = numberOrNull(device.power && device.power.usage_milliwatts);
        return mw === null ? null : mw / 1000;
      },
    },
    {
      id: "fan",
      title: "Fan",
      subtitle: "Fan speed",
      unit: "%",
      domain: [0, 100],
      value(device) {
        return numberOrNull(device.temperature && device.temperature.fan_speed_percent);
      },
    },
  ];

  const colors = ["#1f6fdb", "#dc5f00", "#179a63", "#8b5cf6", "#c2415d", "#0f8b8d", "#6d7d00", "#9b4d96"];
  const chartMap = new Map(metrics.map((metric) => [metric.id, metric]));
  const root = document.querySelector(".app-shell");

  const dom = {
    statusDot: document.querySelector("[data-status-dot]"),
    statusLabel: document.querySelector("[data-status-label]"),
    deviceList: document.querySelector("[data-device-list]"),
    selectAll: document.querySelector("[data-select-all]"),
    refreshEnabled: document.querySelector("[data-refresh-enabled]"),
    refreshInterval: document.querySelector("[data-refresh-interval]"),
    timeWindow: document.querySelector("[data-time-window]"),
    refreshNow: document.querySelector("[data-refresh-now]"),
    summary: document.querySelector("[data-summary]"),
    charts: document.querySelector("[data-charts]"),
  };

  const params = new URLSearchParams(window.location.search);
  const initialGPUParam = params.get("gpu");
  const state = {
    samples: [],
    devices: new Map(),
    selectedIds: new Set(initialGPUParam ? initialGPUParam.split(",").filter(Boolean) : []),
    explicitSelection: Boolean(initialGPUParam),
    focusedChart: chartMap.has(params.get("chart")) ? params.get("chart") : null,
    hoverTime: null,
    refreshEnabled: true,
    refreshInterval: 5000,
    timeWindow: "all",
    zoom: null,
    timer: 0,
    fetching: false,
  };

  const charts = new Map();
  const tooltip = document.createElement("div");
  tooltip.className = "tooltip";
  document.body.appendChild(tooltip);

  renderChartShells();
  bindControls();
  fetchSnapshot();
  scheduleRefresh();
  window.addEventListener("resize", renderAll);
  window.addEventListener("beforeunload", (event) => {
    if (state.samples.length === 0) return;
    event.preventDefault();
    event.returnValue = "";
  });

  function bindControls() {
    dom.selectAll.addEventListener("click", () => {
      state.explicitSelection = false;
      state.selectedIds = new Set(state.devices.keys());
      updateURL();
      renderAll();
    });

    dom.refreshEnabled.addEventListener("change", () => {
      state.refreshEnabled = dom.refreshEnabled.value === "on";
      scheduleRefresh();
      setStatus(state.refreshEnabled ? "ok" : "warn", state.refreshEnabled ? "Auto refresh on" : "Auto refresh paused");
    });

    dom.refreshInterval.addEventListener("change", () => {
      state.refreshInterval = Number(dom.refreshInterval.value);
      scheduleRefresh();
    });

    dom.timeWindow.addEventListener("change", () => {
      state.timeWindow = dom.timeWindow.value;
      state.zoom = null;
      renderAll();
    });

    dom.refreshNow.addEventListener("click", fetchSnapshot);
  }

  function renderChartShells() {
    dom.charts.innerHTML = "";
    for (const metric of metrics) {
      const card = document.createElement("article");
      card.className = "chart-card";
      card.dataset.chart = metric.id;
      if (state.focusedChart === metric.id) card.classList.add("is-fullscreen");

      const header = document.createElement("div");
      header.className = "chart-header";
      header.innerHTML = `
        <div class="chart-title">
          <h2>${escapeHTML(metric.title)}</h2>
          <p>${escapeHTML(metric.subtitle)}</p>
        </div>
        <div class="chart-actions">
          <button type="button" data-reset-zoom>Reset zoom</button>
          <button type="button" data-toggle-fullscreen>${state.focusedChart === metric.id ? "Close" : "Fullscreen"}</button>
        </div>
      `;

      const wrap = document.createElement("div");
      wrap.className = "chart-wrap";
      const canvas = document.createElement("canvas");
      wrap.appendChild(canvas);
      card.append(header, wrap);
      dom.charts.appendChild(card);

      charts.set(metric.id, { metric, card, canvas, wrap });

      header.querySelector("[data-reset-zoom]").addEventListener("click", () => {
        state.zoom = null;
        renderAll();
      });
      header.querySelector("[data-toggle-fullscreen]").addEventListener("click", () => {
        state.focusedChart = state.focusedChart === metric.id ? null : metric.id;
        updateURL();
        renderChartShells();
        renderAll();
      });
      canvas.addEventListener("pointermove", (event) => handlePointerMove(event, metric.id));
      canvas.addEventListener("pointerleave", () => {
        state.hoverTime = null;
        tooltip.classList.remove("is-visible");
        renderAll();
      });
      canvas.addEventListener("wheel", (event) => handleWheel(event, metric.id), { passive: false });
    }
  }

  async function fetchSnapshot() {
    if (state.fetching) return;
    state.fetching = true;
    setStatus("warn", "Refreshing...");
    try {
      const response = await fetch("/api/gpus", { headers: { Accept: "application/json" } });
      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.error || `Request failed with status ${response.status}`);
      }
      addSample(payload);
      setStatus("ok", `Updated ${formatTime(Date.now())}`);
    } catch (error) {
      setStatus("error", error.message);
    } finally {
      state.fetching = false;
      scheduleRefresh();
    }
  }

  function addSample(payload) {
    const collectedAt = Date.parse(payload.collected_at);
    const devices = Array.isArray(payload.snapshot && payload.snapshot.devices) ? payload.snapshot.devices : [];
    const normalized = new Map();

    for (const device of devices) {
      const id = deviceId(device);
      normalized.set(id, device);
      if (!state.devices.has(id)) {
        state.devices.set(id, {
          id,
          label: deviceLabel(device),
          color: colors[state.devices.size % colors.length],
        });
      }
    }

    if (!state.explicitSelection) {
      state.selectedIds = new Set(state.devices.keys());
    } else {
      const available = new Set(state.devices.keys());
      state.selectedIds = new Set([...state.selectedIds].filter((id) => available.has(id)));
      if (state.selectedIds.size === 0 && state.devices.size > 0) {
        state.explicitSelection = false;
        state.selectedIds = new Set(state.devices.keys());
      }
    }

    state.samples.push({ time: Number.isFinite(collectedAt) ? collectedAt : Date.now(), devices: normalized });
    renderAll();
  }

  function scheduleRefresh() {
    window.clearTimeout(state.timer);
    if (!state.refreshEnabled) return;
    state.timer = window.setTimeout(fetchSnapshot, state.refreshInterval);
  }

  function renderAll() {
    renderDevices();
    renderSummary();
    for (const chart of charts.values()) {
      drawChart(chart);
    }
  }

  function renderDevices() {
    if (state.devices.size === 0) {
      dom.deviceList.innerHTML = '<span class="muted">Waiting for devices...</span>';
      return;
    }

    dom.deviceList.innerHTML = "";
    for (const device of state.devices.values()) {
      const label = document.createElement("label");
      label.className = "device-chip";

      const input = document.createElement("input");
      input.type = "checkbox";
      input.checked = state.selectedIds.has(device.id);
      input.addEventListener("change", () => {
        state.explicitSelection = true;
        if (input.checked) {
          state.selectedIds.add(device.id);
        } else {
          state.selectedIds.delete(device.id);
        }
        if (state.selectedIds.size === 0) {
          state.selectedIds.add(device.id);
          input.checked = true;
        }
        updateURL();
        renderAll();
      });

      const text = document.createElement("span");
      text.title = device.label;
      text.textContent = device.label;
      label.append(input, text);
      dom.deviceList.appendChild(label);
    }
  }

  function renderSummary() {
    const latest = state.samples[state.samples.length - 1];
    if (!latest) {
      dom.summary.innerHTML = "";
      return;
    }

    dom.summary.innerHTML = "";
    for (const deviceInfo of selectedDevices()) {
      const device = latest.devices.get(deviceInfo.id);
      if (!device) continue;
      const card = document.createElement("article");
      card.className = "summary-card";
      card.innerHTML = `
        <p class="summary-title" title="${escapeHTML(deviceInfo.label)}">${escapeHTML(deviceInfo.label)}</p>
        <div class="summary-values">
          <span>GPU ${formatMetric(chartMap.get("gpu-util"), device)}</span>
          <span>MEM ${formatMetric(chartMap.get("mem-util"), device)}</span>
          <span>Temp ${formatMetric(chartMap.get("temp"), device)}</span>
          <span>Power ${formatMetric(chartMap.get("power"), device)}</span>
        </div>
      `;
      dom.summary.appendChild(card);
    }
  }

  function drawChart(chart) {
    const { canvas, metric } = chart;
    const rect = canvas.getBoundingClientRect();
    const ratio = window.devicePixelRatio || 1;
    const width = Math.max(320, Math.floor(rect.width));
    const height = Math.max(180, Math.floor(rect.height));
    if (canvas.width !== Math.floor(width * ratio) || canvas.height !== Math.floor(height * ratio)) {
      canvas.width = Math.floor(width * ratio);
      canvas.height = Math.floor(height * ratio);
    }

    const ctx = canvas.getContext("2d");
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    ctx.clearRect(0, 0, width, height);

    const padding = { top: 18, right: 18, bottom: 34, left: 56 };
    const plot = {
      x: padding.left,
      y: padding.top,
      w: width - padding.left - padding.right,
      h: height - padding.top - padding.bottom,
    };

    drawBackground(ctx, plot, width, height);
    const visibleSamples = samplesInRange();
    if (visibleSamples.length === 0 || selectedDevices().length === 0) {
      drawEmpty(ctx, plot, "Waiting for samples");
      return;
    }

    const xRange = currentRange();
    const yRange = yDomain(metric, visibleSamples);
    drawGrid(ctx, plot, xRange, yRange, metric.unit);

    for (const device of selectedDevices()) {
      drawSeries(ctx, plot, xRange, yRange, visibleSamples, metric, device);
    }

    if (state.hoverTime !== null) {
      drawHover(ctx, plot, xRange, yRange, visibleSamples, metric);
    }
  }

  function drawBackground(ctx, plot, width, height) {
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, width, height);
    ctx.fillStyle = "#fbfdff";
    ctx.fillRect(plot.x, plot.y, plot.w, plot.h);
    ctx.strokeStyle = "#dce5ef";
    ctx.lineWidth = 1;
    ctx.strokeRect(plot.x, plot.y, plot.w, plot.h);
  }

  function drawEmpty(ctx, plot, text) {
    ctx.fillStyle = "#6d7d91";
    ctx.font = "14px Inter, system-ui, sans-serif";
    ctx.textAlign = "center";
    ctx.fillText(text, plot.x + plot.w / 2, plot.y + plot.h / 2);
  }

  function drawGrid(ctx, plot, xRange, yRange, unit) {
    ctx.save();
    ctx.strokeStyle = "#e8eef5";
    ctx.fillStyle = "#68788d";
    ctx.font = "12px Inter, system-ui, sans-serif";
    ctx.lineWidth = 1;

    for (let i = 0; i <= 4; i += 1) {
      const y = plot.y + (plot.h * i) / 4;
      ctx.beginPath();
      ctx.moveTo(plot.x, y);
      ctx.lineTo(plot.x + plot.w, y);
      ctx.stroke();

      const value = yRange.max - ((yRange.max - yRange.min) * i) / 4;
      ctx.textAlign = "right";
      ctx.fillText(`${formatNumber(value)}${unit ? ` ${unit}` : ""}`, plot.x - 8, y + 4);
    }

    for (let i = 0; i <= 4; i += 1) {
      const x = plot.x + (plot.w * i) / 4;
      ctx.beginPath();
      ctx.moveTo(x, plot.y);
      ctx.lineTo(x, plot.y + plot.h);
      ctx.stroke();

      const value = xRange.start + ((xRange.end - xRange.start) * i) / 4;
      ctx.textAlign = i === 0 ? "left" : i === 4 ? "right" : "center";
      ctx.fillText(formatTime(value), x, plot.y + plot.h + 22);
    }
    ctx.restore();
  }

  function drawSeries(ctx, plot, xRange, yRange, samples, metric, deviceInfo) {
    const points = samples.map((sample) => {
      const device = sample.devices.get(deviceInfo.id);
      const value = device ? metric.value(device) : null;
      return {
        x: xScale(sample.time, xRange, plot),
        y: value === null ? null : yScale(value, yRange, plot),
        value,
      };
    });

    ctx.save();
    ctx.strokeStyle = deviceInfo.color;
    ctx.lineWidth = 2;
    ctx.lineJoin = "round";
    ctx.lineCap = "round";

    let drawing = false;
    for (const point of points) {
      if (point.value === null) {
        drawing = false;
        continue;
      }
      if (!drawing) {
        ctx.beginPath();
        ctx.moveTo(point.x, point.y);
        drawing = true;
      } else {
        ctx.lineTo(point.x, point.y);
      }
    }
    if (drawing) ctx.stroke();
    ctx.restore();
  }

  function drawHover(ctx, plot, xRange, yRange, samples, metric) {
    const sample = nearestSample(samples, state.hoverTime);
    if (!sample) return;
    const x = xScale(sample.time, xRange, plot);

    ctx.save();
    ctx.strokeStyle = "#25364f";
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(x, plot.y);
    ctx.lineTo(x, plot.y + plot.h);
    ctx.stroke();
    ctx.setLineDash([]);

    for (const deviceInfo of selectedDevices()) {
      const device = sample.devices.get(deviceInfo.id);
      const value = device ? metric.value(device) : null;
      if (value === null) continue;
      const y = yScale(value, yRange, plot);
      ctx.fillStyle = "#ffffff";
      ctx.strokeStyle = deviceInfo.color;
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.arc(x, y, 4, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
    }
    ctx.restore();
  }

  function handlePointerMove(event, metricID) {
    const chart = charts.get(metricID);
    const rect = chart.canvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const plot = plotFor(chart.canvas);
    const range = currentRange();
    const time = range.start + ((x - plot.x) / plot.w) * (range.end - range.start);
    const sample = nearestSample(samplesInRange(), time);
    if (!sample) return;
    state.hoverTime = sample.time;
    showTooltip(event.clientX, event.clientY, chart.metric, sample);
    renderAll();
  }

  function handleWheel(event, metricID) {
    const chart = charts.get(metricID);
    const rect = chart.canvas.getBoundingClientRect();
    const plot = plotFor(chart.canvas);
    const x = event.clientX - rect.left;
    if (x < plot.x || x > plot.x + plot.w) return;
    event.preventDefault();

    const full = fullRange();
    if (!full) return;
    const current = currentRange();
    const anchor = current.start + ((x - plot.x) / plot.w) * (current.end - current.start);
    const factor = event.deltaY < 0 ? 0.82 : 1.22;
    const minSpan = Math.max(10000, state.refreshInterval * 2);
    let span = Math.max(minSpan, (current.end - current.start) * factor);
    span = Math.min(span, full.end - full.start || minSpan);
    let start = anchor - (anchor - current.start) * factor;
    let end = start + span;
    if (start < full.start) {
      start = full.start;
      end = start + span;
    }
    if (end > full.end) {
      end = full.end;
      start = end - span;
    }
    state.zoom = { start, end };
    renderAll();
  }

  function showTooltip(clientX, clientY, metric, sample) {
    const rows = selectedDevices()
      .map((deviceInfo) => {
        const device = sample.devices.get(deviceInfo.id);
        const value = device ? formatMetric(metric, device) : "n/a";
        return `
          <div class="tooltip-row">
            <span class="tooltip-swatch" style="background:${deviceInfo.color}"></span>
            <span class="tooltip-name">${escapeHTML(deviceInfo.label)}</span>
            <span>${escapeHTML(value)}</span>
          </div>
        `;
      })
      .join("");

    tooltip.innerHTML = `<strong>${escapeHTML(metric.title)} at ${escapeHTML(formatDateTime(sample.time))}</strong>${rows}`;
    tooltip.classList.add("is-visible");
    const x = Math.min(clientX + 16, window.innerWidth - tooltip.offsetWidth - 12);
    const y = Math.min(clientY + 16, window.innerHeight - tooltip.offsetHeight - 12);
    tooltip.style.transform = `translate(${Math.max(12, x)}px, ${Math.max(12, y)}px)`;
  }

  function updateURL() {
    const next = new URLSearchParams(window.location.search);
    const allSelected = state.selectedIds.size === state.devices.size;
    if (!state.explicitSelection || allSelected) {
      next.delete("gpu");
    } else {
      next.set("gpu", [...state.selectedIds].join(","));
    }
    if (state.focusedChart) {
      next.set("chart", state.focusedChart);
    } else {
      next.delete("chart");
    }
    const query = next.toString();
    window.history.replaceState(null, "", query ? `${window.location.pathname}?${query}` : window.location.pathname);
  }

  function samplesInRange() {
    const range = currentRange();
    return state.samples.filter((sample) => sample.time >= range.start && sample.time <= range.end);
  }

  function currentRange() {
    if (state.zoom) return state.zoom;
    const full = fullRange();
    if (!full) {
      const now = Date.now();
      return { start: now - 60000, end: now };
    }
    if (state.timeWindow === "all") return full;
    const windowSize = Number(state.timeWindow);
    return { start: Math.max(full.start, full.end - windowSize), end: full.end };
  }

  function fullRange() {
    if (state.samples.length === 0) return null;
    const first = state.samples[0].time;
    const last = state.samples[state.samples.length - 1].time;
    if (first === last) {
      return { start: first - 30000, end: last + 30000 };
    }
    return { start: first, end: last };
  }

  function yDomain(metric, samples) {
    if (Array.isArray(metric.domain)) return { min: metric.domain[0], max: metric.domain[1] };
    let min = Infinity;
    let max = -Infinity;
    for (const sample of samples) {
      for (const deviceInfo of selectedDevices()) {
        const device = sample.devices.get(deviceInfo.id);
        const value = device ? metric.value(device) : null;
        if (value === null) continue;
        min = Math.min(min, value);
        max = Math.max(max, value);
      }
    }
    if (!Number.isFinite(min) || !Number.isFinite(max)) return { min: 0, max: 1 };
    if (min === max) {
      const pad = min === 0 ? 1 : Math.abs(min) * 0.1;
      return { min: Math.max(0, min - pad), max: max + pad };
    }
    const pad = (max - min) * 0.12;
    return { min: Math.max(0, min - pad), max: max + pad };
  }

  function selectedDevices() {
    return [...state.devices.values()].filter((device) => state.selectedIds.has(device.id));
  }

  function nearestSample(samples, time) {
    if (samples.length === 0) return null;
    let nearest = samples[0];
    let distance = Math.abs(samples[0].time - time);
    for (const sample of samples) {
      const nextDistance = Math.abs(sample.time - time);
      if (nextDistance < distance) {
        nearest = sample;
        distance = nextDistance;
      }
    }
    return nearest;
  }

  function plotFor(canvas) {
    const rect = canvas.getBoundingClientRect();
    return { x: 56, y: 18, w: Math.max(1, rect.width - 74), h: Math.max(1, rect.height - 52) };
  }

  function xScale(value, range, plot) {
    return plot.x + ((value - range.start) / (range.end - range.start || 1)) * plot.w;
  }

  function yScale(value, range, plot) {
    return plot.y + plot.h - ((value - range.min) / (range.max - range.min || 1)) * plot.h;
  }

  function formatMetric(metric, device) {
    if (metric.detail) return metric.detail(device);
    const value = metric.value(device);
    if (value === null) return "n/a";
    return `${formatNumber(value)} ${metric.unit}`;
  }

  function formatNumber(value) {
    if (Math.abs(value) >= 100) return value.toFixed(0);
    if (Math.abs(value) >= 10) return value.toFixed(1);
    return value.toFixed(2);
  }

  function formatTime(value) {
    return new Intl.DateTimeFormat(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }).format(new Date(value));
  }

  function formatDateTime(value) {
    return new Intl.DateTimeFormat(undefined, {
      month: "short",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }).format(new Date(value));
  }

  function bytesToGiB(value) {
    return Number(value || 0) / 1024 / 1024 / 1024;
  }

  function numberOrNull(value) {
    return typeof value === "number" && Number.isFinite(value) ? value : null;
  }

  function deviceId(device) {
    if (device.uuid) return String(device.uuid);
    if (device.index !== undefined && device.index !== null) return String(device.index);
    return device.name || "unknown";
  }

  function deviceLabel(device) {
    const index = device.index !== undefined && device.index !== null ? `GPU ${device.index}` : "GPU";
    const name = device.name || "Unknown device";
    const uuid = device.uuid || "no UUID";
    return `${index} - ${name} - ${uuid}`;
  }

  function setStatus(kind, text) {
    dom.statusDot.className = `status-dot is-${kind}`;
    dom.statusLabel.textContent = text;
  }

  function escapeHTML(value) {
    return String(value).replace(/[&<>"']/g, (char) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    }[char]));
  }
})();
