(() => {
  const metrics = [
    {
      id: "memory",
      title: "Memory usage",
      unit: "GiB",
      domain: "auto",
      value(device) {
        return device.memory ? bytesToGiB(device.memory.used_bytes) : null;
      },
      tooltip(device) {
        if (!device.memory) return "n/a";
        return `${formatNumber(bytesToMiB(device.memory.used_bytes))} MB`;
      },
      detail(device) {
        if (!device.memory) return "n/a";
        return `${bytesToGiB(device.memory.used_bytes).toFixed(2)}/${bytesToGiB(device.memory.total_bytes).toFixed(2)} GiB`;
      },
    },
    {
      id: "gpu-util",
      title: "GPU utilization (%)",
      unit: "%",
      domain: [0, 100],
      value(device) {
        return numberOrNull(device.utilization && device.utilization.gpu_percent);
      },
    },
    {
      id: "mem-util",
      title: "Memory utilization (%)",
      unit: "%",
      domain: [0, 100],
      value(device) {
        return numberOrNull(device.utilization && device.utilization.memory_percent);
      },
    },
    {
      id: "temp",
      title: "Temperature",
      unit: "C",
      domain: "auto",
      value(device) {
        return numberOrNull(device.temperature && device.temperature.gpu_celsius);
      },
    },
    {
      id: "power",
      title: "Power usage",
      unit: "W",
      domain: "auto",
      value(device) {
        const mw = numberOrNull(device.power && device.power.usage_milliwatts);
        return mw === null ? null : mw / 1000;
      },
    },
    {
      id: "fan",
      title: "Fan speed",
      unit: "%",
      domain: [0, 100],
      value(device) {
        return numberOrNull(device.temperature && device.temperature.fan_speed_percent);
      },
    },
  ];

  const summaryChartID = "summary";
  const chartOptions = [{ id: summaryChartID, title: "Summary" }, ...metrics];
  const defaultChartIds = [summaryChartID, "memory", "gpu-util", "mem-util", "temp"];
  const colors = ["#1f6fdb", "#dc5f00", "#179a63", "#8b5cf6", "#c2415d", "#0f8b8d", "#6d7d00", "#9b4d96"];
  const aggregateChartWindow = 60 * 60 * 1000;
  const minPixelsPerChartPoint = 3;
  const chartMap = new Map(metrics.map((metric) => [metric.id, metric]));
  const chartOptionMap = new Map(chartOptions.map((option) => [option.id, option]));
  const chartParamMap = new Map(chartOptions.map((option, index) => [option.id, String(index)]));

  const dom = {
    appShell: document.querySelector("[data-branding]"),
    brandReset: document.querySelector("[data-brand-reset]"),
    dropdowns: [...document.querySelectorAll("[data-dropdown]")],
    gpuDropdown: document.querySelector('[data-dropdown="gpu"]'),
    chartDropdown: document.querySelector('[data-dropdown="charts"]'),
    timeDropdown: document.querySelector('[data-dropdown="time"]'),
    refreshDropdown: document.querySelector('[data-dropdown="refresh"]'),
    gpuTrigger: document.querySelector("[data-gpu-trigger]"),
    gpuSummary: document.querySelector("[data-gpu-summary]"),
    chartTrigger: document.querySelector("[data-chart-trigger]"),
    chartSummary: document.querySelector("[data-chart-summary]"),
    chartList: document.querySelector("[data-chart-list]"),
    hostInfoTrigger: document.querySelector("[data-host-info-trigger]"),
    hostModal: document.querySelector("[data-host-modal]"),
    hostModalClose: document.querySelector("[data-host-modal-close]"),
    hostModalTitle: document.querySelector("[data-host-modal-title]"),
    hostModalHostName: document.querySelector("[data-host-modal-host-name]"),
    hostModalDriverVersion: document.querySelector("[data-host-modal-driver-version]"),
    hostModalCUDAVersion: document.querySelector("[data-host-modal-cuda-version]"),
    hostModalGPUCount: document.querySelector("[data-host-modal-gpu-count]"),
    hostModalMemoryTotal: document.querySelector("[data-host-modal-memory-total]"),
    hostModalMemoryUsed: document.querySelector("[data-host-modal-memory-used]"),
    hostModalMemoryFree: document.querySelector("[data-host-modal-memory-free]"),
    hostModalGPUUtilization: document.querySelector("[data-host-modal-gpu-utilization]"),
    timeMinutes: document.querySelector("[data-time-minutes]"),
    statusDot: document.querySelector("[data-status-dot]"),
    statusTooltip: document.querySelector("[data-status-tooltip]"),
    deviceList: document.querySelector("[data-device-list]"),
    selectAll: document.querySelector("[data-select-all]"),
    clearAll: document.querySelector("[data-clear-all]"),
    hostControl: document.querySelector("[data-host-control]"),
    chartSelectAll: document.querySelector("[data-chart-select-all]"),
    chartClearAll: document.querySelector("[data-chart-clear-all]"),
    refreshInterval: document.querySelector("[data-refresh-interval]"),
    refreshNow: document.querySelector("[data-refresh-now]"),
    timeInputLabel: document.querySelector(".time-input-label"),
    refreshInputLabel: document.querySelector(".refresh-input-label"),
    timePresets: [...document.querySelectorAll("[data-time-preset]")],
    refreshPresets: [...document.querySelectorAll("[data-refresh-preset]")],
    summary: document.querySelector("[data-summary]"),
    charts: document.querySelector("[data-charts]"),
  };

  const params = new URLSearchParams(window.location.search);
  const hostOptions = parseHostConfig(dom.appShell ? dom.appShell.dataset.hosts : "[]");
  const defaultHostIndex = defaultHostOption(hostOptions);
  const basePageTitle = document.title;
  const initialGPUParam = params.get("gpu");
  const initialChartIds = parseChartParam(params.get("charts"));
  const initialFocusedChart = parseChartIDParam(params.get("chart"));
  const state = {
    samples: [],
    devices: new Map(),
    selectedIds: new Set(parseGPUParam(initialGPUParam)),
    selectedChartIds: new Set(initialChartIds === null ? defaultChartIds : initialChartIds),
    hostIndex: parseHostParam(params.get("host"), hostOptions, defaultHostIndex),
    explicitSelection: Boolean(initialGPUParam),
    focusedChart: initialFocusedChart,
    hoverTime: null,
    refreshInterval: 1000,
    timeWindow: 300000,
    timer: 0,
    fetching: false,
    requestID: 0,
    lastUpdate: null,
  };
  if (state.focusedChart === summaryChartID) {
    state.focusedChart = null;
  }
  if (state.focusedChart && !state.selectedChartIds.has(state.focusedChart)) {
    state.focusedChart = null;
  }

  const charts = new Map();
  const tooltip = document.createElement("div");
  tooltip.className = "tooltip";
  document.body.appendChild(tooltip);

  updatePageTitle();
  renderHostControl();
  renderChartShells();
  bindControls();
  renderChartSelector();
  renderControlSummaries();
  updateURL();
  fetchSnapshot();
  scheduleRefresh();
  window.addEventListener("resize", renderAll);
  window.addEventListener("beforeunload", (event) => {
    if (state.samples.length === 0) return;
    event.preventDefault();
    event.returnValue = "";
  });

  function bindControls() {
    dom.brandReset.addEventListener("click", (event) => {
      event.preventDefault();
      resetToDefaults();
    });

    dom.gpuTrigger.addEventListener("click", () => toggleDropdown(dom.gpuDropdown));
    dom.chartTrigger.addEventListener("click", () => toggleDropdown(dom.chartDropdown));
    dom.hostInfoTrigger.addEventListener("click", openHostModal);
    dom.hostModalClose.addEventListener("click", closeHostModal);
    dom.hostModal.addEventListener("click", (event) => {
      if (event.target === dom.hostModal) {
        closeHostModal();
      }
    });
    dom.timeInputLabel.addEventListener("click", () => focusTimeInput());
    dom.timeMinutes.addEventListener("focus", () => openDropdown(dom.timeDropdown, false));
    dom.timeMinutes.addEventListener("change", applyTimeInput);
    dom.timeMinutes.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        applyTimeInput();
        dom.timeMinutes.blur();
        closeDropdowns();
      }
    });
    dom.refreshInputLabel.addEventListener("click", () => focusRefreshInput());
    dom.refreshInterval.addEventListener("focus", () => openDropdown(dom.refreshDropdown, false));
    dom.refreshInterval.addEventListener("change", applyRefreshInput);
    dom.refreshInterval.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        applyRefreshInput();
        dom.refreshInterval.blur();
        closeDropdowns();
      }
    });
    dom.statusDot.addEventListener("mouseenter", updateStatusTooltipPosition);
    dom.statusDot.addEventListener("mousemove", updateStatusTooltipPosition);
    dom.statusDot.addEventListener("focus", () => {
      const rect = dom.statusDot.getBoundingClientRect();
      updateStatusTooltipPosition({
        clientX: rect.left + rect.width / 2,
        clientY: rect.bottom,
      });
    });

    document.addEventListener("click", (event) => {
      if (dom.dropdowns.some((dropdown) => dropdown.contains(event.target))) return;
      closeDropdowns();
    });

    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape") {
        closeDropdowns();
        closeHostModal();
      }
    });

    dom.selectAll.addEventListener("click", () => {
      state.explicitSelection = false;
      state.selectedIds = new Set(state.devices.keys());
      updateURL();
      renderAll();
    });

    dom.clearAll.addEventListener("click", () => {
      state.explicitSelection = true;
      state.selectedIds = new Set();
      updateURL();
      renderAll();
    });

    dom.chartSelectAll.addEventListener("click", () => {
      state.selectedChartIds = new Set(chartOptions.map((option) => option.id));
      updateURL();
      renderChartSelector();
      renderChartShells();
      renderAll();
    });

    dom.chartClearAll.addEventListener("click", () => {
      state.selectedChartIds = new Set();
      state.focusedChart = null;
      updateURL();
      renderChartSelector();
      renderChartShells();
      renderAll();
    });

    dom.timePresets.forEach((button) => {
      button.addEventListener("click", () => {
        const minutes = Number(button.dataset.timePreset);
        setTimeWindowMinutes(minutes);
        closeDropdowns();
        renderAll();
      });
    });

    dom.refreshPresets.forEach((button) => {
      button.addEventListener("click", () => {
        const seconds = Number(button.dataset.refreshPreset);
        setRefreshIntervalSeconds(seconds);
        closeDropdowns();
      });
    });

    dom.refreshNow.addEventListener("click", fetchSnapshot);
  }

  function renderChartShells() {
    dom.charts.innerHTML = "";
    charts.clear();
    dom.appShell.classList.toggle("is-chart-focused", Boolean(state.focusedChart));
    dom.appShell.classList.toggle("is-summary-hidden", !state.selectedChartIds.has(summaryChartID));
    const selectedMetrics = metrics.filter((metric) => {
      if (!state.selectedChartIds.has(metric.id)) return false;
      return !state.focusedChart || state.focusedChart === metric.id;
    });
    dom.charts.style.setProperty("--chart-rows", String(Math.max(1, Math.ceil(selectedMetrics.length / 2))));
    dom.charts.dataset.count = String(selectedMetrics.length);
    if (selectedMetrics.length === 0) {
      dom.charts.innerHTML = '<div class="empty-charts">Select required charts in navigation bar</div>';
      return;
    }

    for (const metric of selectedMetrics) {
      const card = document.createElement("article");
      card.className = "chart-card";
      card.dataset.chart = metric.id;
      if (state.focusedChart === metric.id) card.classList.add("is-fullscreen");

      const header = document.createElement("div");
      header.className = "chart-header";
      header.innerHTML = `
        <div class="chart-title">
          <h2>${escapeHTML(metric.title)}</h2>
        </div>
        <div class="chart-actions">
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
    }
  }

  async function fetchSnapshot() {
    if (state.fetching) return;
    const requestID = state.requestID + 1;
    state.requestID = requestID;
    state.fetching = true;
    try {
      const response = await fetch(gpuAPIURL(), { headers: { Accept: "application/json" } });
      const payload = await response.json();
      if (requestID !== state.requestID) return;
      if (!response.ok) {
        throw new Error(payload.error || `Request failed with status ${response.status}`);
      }
      addSample(payload);
      state.lastUpdate = Date.now();
      updateStatus();
    } catch (error) {
      if (requestID === state.requestID) {
        updateStatus("error", error.message);
      }
    } finally {
      if (requestID === state.requestID) {
        state.fetching = false;
        scheduleRefresh();
      }
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
          index: device.index,
          label: deviceLabel(device),
          shortLabel: shortDeviceLabel(device),
          color: colors[state.devices.size % colors.length],
          aliases: deviceAliases(device, id),
        });
      }
    }

    if (!state.explicitSelection) {
      state.selectedIds = new Set(state.devices.keys());
    } else {
      const available = new Set(state.devices.keys());
      state.selectedIds = resolveSelectedDeviceIds(available);
    }

    state.samples.push({
      time: Number.isFinite(collectedAt) ? collectedAt : Date.now(),
      devices: normalized,
      system: payload.snapshot && payload.snapshot.system ? payload.snapshot.system : {},
    });
    renderAll();
  }

  function scheduleRefresh() {
    window.clearTimeout(state.timer);
    if (state.refreshInterval <= 0) return;
    state.timer = window.setTimeout(fetchSnapshot, state.refreshInterval);
  }

  function renderAll() {
    renderDevices();
    renderControlSummaries();
    renderSummary();
    if (!dom.hostModal.hidden) {
      renderHostModal();
    }
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
      label.className = "device-option";

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
        updateURL();
        renderAll();
      });

      const dot = document.createElement("span");
      dot.className = "color-dot";
      dot.style.background = device.color;

      const text = document.createElement("span");
      text.title = device.label;
      text.textContent = device.label;

      label.append(input, dot, text);
      dom.deviceList.appendChild(label);
    }
  }

  function renderChartSelector() {
    dom.chartList.innerHTML = "";
    for (const option of chartOptions) {
      const label = document.createElement("label");
      label.className = "chart-option";

      const input = document.createElement("input");
      input.type = "checkbox";
      input.checked = state.selectedChartIds.has(option.id);
      input.addEventListener("change", () => {
        if (input.checked) {
          state.selectedChartIds.add(option.id);
        } else {
          state.selectedChartIds.delete(option.id);
        }
        if (state.focusedChart && !state.selectedChartIds.has(state.focusedChart)) {
          state.focusedChart = null;
        }
        updateURL();
        renderChartShells();
        renderAll();
      });

      const text = document.createElement("span");
      text.textContent = option.title;
      label.append(input, text);
      dom.chartList.appendChild(label);
    }
  }

  function renderControlSummaries() {
    renderGPUSummary();
    renderChartSummary();
  }

  function renderHostControl() {
    if (!dom.hostControl) return;
    dom.hostControl.innerHTML = "";

    if (hostOptions.length === 0) {
      dom.hostControl.hidden = true;
      return;
    }
    dom.hostControl.hidden = false;

    const label = document.createElement("label");
    label.className = "host-select-label";

    const select = document.createElement("select");
    select.setAttribute("aria-label", "GPU host");
    for (const host of hostOptions) {
      const option = document.createElement("option");
      option.value = String(host.index);
      option.textContent = host.name;
      select.appendChild(option);
    }
    select.value = String(state.hostIndex);
    select.addEventListener("change", () => selectHost(Number(select.value)));

    label.append(select);
    dom.hostControl.appendChild(label);
  }

  function openHostModal() {
    closeDropdowns();
    renderHostModal();
    dom.hostModal.hidden = false;
    dom.hostModalClose.focus();
  }

  function closeHostModal() {
    dom.hostModal.hidden = true;
  }

  function renderHostModal() {
    const system = latestSystemInfo();
    const memory = hostMemorySummary();
    dom.hostModalTitle.textContent = currentHostName();
    dom.hostModalHostName.textContent = textOrNA(system.host_name || currentHostAddress());
    dom.hostModalDriverVersion.textContent = textOrNA(system.driver_version);
    dom.hostModalCUDAVersion.textContent = textOrNA(system.cuda_driver_version);
    dom.hostModalGPUCount.textContent = String(memory.gpuCount);
    dom.hostModalMemoryTotal.innerHTML = formatHostMemoryHTML(memory.totalBytes, memory.hasMemory);
    dom.hostModalMemoryUsed.innerHTML = formatHostMemoryPercentHTML(memory.usedBytes, memory);
    dom.hostModalMemoryFree.innerHTML = formatHostMemoryPercentHTML(memory.freeBytes, memory);
    dom.hostModalGPUUtilization.innerHTML = formatHostPercentHTML(hostAverageGPUUtilization());
  }

  function updatePageTitle() {
    document.title = `${basePageTitle} - ${currentHostName()}`;
  }

  function currentHostName() {
    if (hostOptions.length === 0) return "local";

    const selectedHost = currentHostOption();
    return selectedHost ? selectedHost.name : "local";
  }

  function currentHostAddress() {
    if (hostOptions.length === 0) return window.location.host || "local";

    const selectedHost = currentHostOption();
    return selectedHost ? selectedHost.hostName : "n/a";
  }

  function currentHostOption() {
    return hostOptions.find((host) => host.index === state.hostIndex);
  }

  function latestSystemInfo() {
    const latest = state.samples[state.samples.length - 1];
    return latest && latest.system ? latest.system : {};
  }

  function hostMemorySummary() {
    const latest = state.samples[state.samples.length - 1];
    const devices = latest ? [...latest.devices.values()] : [];
    return devices.reduce(
      (summary, device) => {
        summary.gpuCount += 1;
        if (!device.memory) {
          return summary;
        }
        summary.totalBytes += Number(device.memory.total_bytes || 0);
        summary.usedBytes += Number(device.memory.used_bytes || 0);
        summary.freeBytes += Number(device.memory.free_bytes || 0);
        summary.hasMemory = true;
        return summary;
      },
      { gpuCount: 0, totalBytes: 0, usedBytes: 0, freeBytes: 0, hasMemory: false },
    );
  }

  function hostAverageGPUUtilization() {
    const latest = state.samples[state.samples.length - 1];
    const devices = latest ? [...latest.devices.values()] : [];
    let total = 0;
    let count = 0;
    for (const device of devices) {
      const value = numberOrNull(device.utilization && device.utilization.gpu_percent);
      if (value === null) continue;
      total += value;
      count += 1;
    }
    return count === 0 ? null : total / count;
  }

  function renderGPUSummary() {
    const total = state.devices.size;
    if (total === 0) {
      dom.gpuSummary.textContent = "Waiting for GPUs";
      return;
    }
    if (!state.explicitSelection || state.selectedIds.size === total) {
      dom.gpuSummary.textContent = `All GPUs (${total})`;
      return;
    }
    if (state.selectedIds.size === 0) {
      dom.gpuSummary.textContent = "No GPUs selected";
      return;
    }

    dom.gpuSummary.innerHTML = "";
    const wrap = document.createElement("span");
    wrap.className = "gpu-summary";
    for (const device of selectedDevices().slice(0, 3)) {
      const badge = document.createElement("span");
      badge.className = "gpu-mini-badge";
      badge.style.background = device.color;
      badge.title = device.label;
      badge.textContent = device.shortLabel;
      wrap.appendChild(badge);
    }
    dom.gpuSummary.appendChild(wrap);
  }

  function renderChartSummary() {
    const count = state.selectedChartIds.size;
    if (count === chartOptions.length) {
      dom.chartSummary.textContent = `All charts (${chartOptions.length})`;
      return;
    }
    if (isDefaultChartSelection()) {
      dom.chartSummary.textContent = "Default charts";
      return;
    }
    if (count === 0) {
      dom.chartSummary.textContent = "No charts selected";
      return;
    }
    dom.chartSummary.textContent = `${count} chart${count === 1 ? "" : "s"}`;
  }

  function renderSummary() {
    if (!state.selectedChartIds.has(summaryChartID) || state.focusedChart) {
      dom.summary.innerHTML = "";
      return;
    }

    const latest = state.samples[state.samples.length - 1];
    if (!latest) {
      dom.summary.innerHTML = "";
      return;
    }

    dom.summary.innerHTML = "";
    for (const deviceInfo of state.devices.values()) {
      const device = latest.devices.get(deviceInfo.id);
      if (!device) continue;
      const isActive = state.selectedIds.has(deviceInfo.id);
      const card = document.createElement("article");
      card.className = `summary-card${isActive ? "" : " is-inactive"}`;
      card.setAttribute("role", "button");
      card.setAttribute("aria-pressed", isActive ? "true" : "false");
      card.tabIndex = 0;
      card.innerHTML = `
        <p class="summary-title" title="${escapeHTML(deviceInfo.label)}">
          <span class="gpu-badge" style="background:${deviceInfo.color}">${escapeHTML(deviceInfo.shortLabel)}</span>
          <span class="device-name">${escapeHTML(deviceName(device))} - ${escapeHTML(device.uuid || "no UUID")}</span>
        </p>
        <div class="summary-values">
          <span>MEM ${formatMetric(chartMap.get("memory"), device)}</span>
          <span>GPU% ${formatMetric(chartMap.get("gpu-util"), device)}</span>
          <span>MEM% ${formatMetric(chartMap.get("mem-util"), device)}</span>
          <span>TEMP ${formatMetric(chartMap.get("temp"), device)}</span>
          <span>POWER ${formatMetric(chartMap.get("power"), device)}</span>
          <span>FAN ${formatMetric(chartMap.get("fan"), device)}</span>
        </div>
      `;
      card.addEventListener("click", () => selectOnlyDevice(deviceInfo.id));
      card.addEventListener("keydown", (event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          selectOnlyDevice(deviceInfo.id);
        }
      });
      dom.summary.appendChild(card);
    }
  }

  function selectOnlyDevice(id) {
    if (state.explicitSelection && state.selectedIds.size === 1 && state.selectedIds.has(id)) {
      state.explicitSelection = false;
      state.selectedIds = new Set(state.devices.keys());
      updateURL();
      closeDropdowns();
      renderAll();
      return;
    }

    state.explicitSelection = true;
    state.selectedIds = new Set([id]);
    updateURL();
    closeDropdowns();
    renderAll();
  }

  function drawChart(chart) {
    const { canvas, metric } = chart;
    const rect = canvas.getBoundingClientRect();
    const ratio = window.devicePixelRatio || 1;
    const width = Math.max(240, Math.floor(rect.width));
    const height = Math.max(140, Math.floor(rect.height));
    if (canvas.width !== Math.floor(width * ratio) || canvas.height !== Math.floor(height * ratio)) {
      canvas.width = Math.floor(width * ratio);
      canvas.height = Math.floor(height * ratio);
    }

    const ctx = canvas.getContext("2d");
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    ctx.clearRect(0, 0, width, height);

    const padding = { top: 14, right: 14, bottom: 30, left: 52 };
    const plot = {
      x: padding.left,
      y: padding.top,
      w: width - padding.left - padding.right,
      h: height - padding.top - padding.bottom,
    };

    drawBackground(ctx, plot, width, height);
    const visibleSamples = samplesInRange();
    if (visibleSamples.length === 0 || selectedDevices().length === 0) {
      drawEmpty(ctx, plot, selectedDevices().length === 0 ? "No GPUs selected" : "Waiting for samples");
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
    ctx.font = "11px Inter, system-ui, sans-serif";
    ctx.lineWidth = 1;

    for (let i = 0; i <= 4; i += 1) {
      const y = plot.y + (plot.h * i) / 4;
      ctx.beginPath();
      ctx.moveTo(plot.x, y);
      ctx.lineTo(plot.x + plot.w, y);
      ctx.stroke();

      const value = yRange.max - ((yRange.max - yRange.min) * i) / 4;
      ctx.textAlign = "right";
      ctx.fillText(`${formatNumber(value)}${unit ? ` ${unit}` : ""}`, plot.x - 7, y + 4);
    }

    for (let i = 0; i <= 4; i += 1) {
      const x = plot.x + (plot.w * i) / 4;
      ctx.beginPath();
      ctx.moveTo(x, plot.y);
      ctx.lineTo(x, plot.y + plot.h);
      ctx.stroke();

      const value = xRange.start + ((xRange.end - xRange.start) * i) / 4;
      ctx.textAlign = i === 0 ? "left" : i === 4 ? "right" : "center";
      ctx.fillText(formatTime(value), x, plot.y + plot.h + 20);
    }
    ctx.restore();
  }

  function drawSeries(ctx, plot, xRange, yRange, samples, metric, deviceInfo) {
    const points = chartPoints(plot, xRange, yRange, samples, metric, deviceInfo);

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

  function chartPoints(plot, xRange, yRange, samples, metric, deviceInfo) {
    if (state.timeWindow < aggregateChartWindow) {
      return samples.map((sample) => samplePoint(sample, plot, xRange, yRange, metric, deviceInfo));
    }

    return thinnedSamples(plot, xRange, samples).map((sample) => samplePoint(sample, plot, xRange, yRange, metric, deviceInfo));
  }

  function samplePoint(sample, plot, xRange, yRange, metric, deviceInfo) {
    const device = sample.devices.get(deviceInfo.id);
    const value = device ? metric.value(device) : null;
    return {
      x: xScale(sample.time, xRange, plot),
      y: value === null ? null : yScale(value, yRange, plot),
      value,
    };
  }

  function thinnedSamples(plot, xRange, samples) {
    if (samples.length <= 2) return samples;

    const firstX = xScale(samples[0].time, xRange, plot);
    const lastX = xScale(samples[samples.length - 1].time, xRange, plot);
    const averageGap = Math.abs(lastX - firstX) / Math.max(1, samples.length - 1);
    const step = Math.max(1, Math.ceil(minPixelsPerChartPoint / Math.max(averageGap, 0.01)));
    if (step === 1) return samples;

    const thinned = [];
    for (let i = 0; i < samples.length; i += step) {
      thinned.push(samples[i]);
    }
    return thinned;
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

  function showTooltip(clientX, clientY, metric, sample) {
    const rows = selectedDevices()
      .map((deviceInfo) => {
        const device = sample.devices.get(deviceInfo.id);
        const value = device ? formatTooltipMetric(metric, device) : "n/a";
        return `
          <div class="tooltip-row">
            <span class="tooltip-swatch" style="background:${deviceInfo.color}"></span>
            <span class="tooltip-name">${escapeHTML(deviceInfo.shortLabel)}</span>
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
    if (hostOptions.length > 0) {
      next.set("host", String(state.hostIndex));
    } else {
      next.delete("host");
    }
    const allSelected = state.selectedIds.size === state.devices.size;
    if (!state.explicitSelection || allSelected) {
      next.delete("gpu");
    } else if (state.selectedIds.size === 0) {
      next.set("gpu", "none");
    } else {
      next.set("gpu", selectedDevices().map((device) => device.id).join(","));
    }
    if (state.focusedChart) {
      next.set("chart", chartIDToParam(state.focusedChart));
    } else {
      next.delete("chart");
    }
    if (isDefaultChartSelection()) {
      next.delete("charts");
    } else if (selectedChartIds().length === 0) {
      next.set("charts", "none");
    } else if (selectedChartIds().length === chartOptions.length) {
      next.set("charts", "all");
    } else {
      next.set("charts", selectedChartIds().map(chartIDToParam).join(","));
    }
    const query = next.toString().replaceAll("%2C", ",");
    window.history.replaceState(null, "", query ? `${window.location.pathname}?${query}` : window.location.pathname);
  }

  function resetToDefaults() {
    const hostChanged = hostOptions.length > 0 && state.hostIndex !== defaultHostIndex;
    state.hostIndex = defaultHostIndex;
    if (hostChanged) clearHostData();
    state.explicitSelection = false;
    state.selectedIds = new Set(state.devices.keys());
    state.selectedChartIds = new Set(defaultChartIds);
    state.focusedChart = null;
    state.hoverTime = null;
    tooltip.classList.remove("is-visible");
    closeDropdowns();
    updatePageTitle();
    renderHostControl();
    updateURL();
    renderChartSelector();
    renderChartShells();
    renderAll();
    if (hostChanged) {
      updateStatus();
      fetchSnapshot();
    }
  }

  function selectHost(index) {
    if (state.hostIndex === index) return;
    state.hostIndex = index;
    state.explicitSelection = false;
    state.selectedIds = new Set();
    clearHostData();
    tooltip.classList.remove("is-visible");
    closeDropdowns();
    updatePageTitle();
    updateURL();
    renderHostControl();
    renderAll();
    updateStatus();
    fetchSnapshot();
  }

  function clearHostData() {
    window.clearTimeout(state.timer);
    state.samples = [];
    state.devices = new Map();
    state.hoverTime = null;
    state.lastUpdate = null;
    state.fetching = false;
    state.requestID += 1;
  }

  function gpuAPIURL() {
    const url = new URL("/api/gpus", window.location.origin);
    if (hostOptions.length > 0) {
      url.searchParams.set("host", String(state.hostIndex));
    }
    return `${url.pathname}${url.search}`;
  }

  function samplesInRange() {
    const range = currentRange();
    return state.samples.filter((sample) => sample.time >= range.start && sample.time <= range.end);
  }

  function currentRange() {
    const end = state.samples.length > 0 ? state.samples[state.samples.length - 1].time : Date.now();
    return { start: end - state.timeWindow, end };
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

  function resolveSelectedDeviceIds(available) {
    const selected = new Set();
    for (const id of state.selectedIds) {
      if (available.has(id)) {
        selected.add(id);
        continue;
      }
      for (const device of state.devices.values()) {
        if (device.aliases.includes(id)) {
          selected.add(device.id);
          break;
        }
      }
    }
    return selected;
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
    return { x: 52, y: 14, w: Math.max(1, rect.width - 66), h: Math.max(1, rect.height - 44) };
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

  function formatTooltipMetric(metric, device) {
    if (metric.tooltip) return metric.tooltip(device);
    return formatMetric(metric, device);
  }

  function formatNumber(value) {
    if (Math.abs(value) >= 100) return value.toFixed(0);
    if (Math.abs(value) >= 10) return value.toFixed(1);
    return value.toFixed(2);
  }

  function formatHostMemoryHTML(summaryBytes, hasMemory) {
    if (!hasMemory) return "n/a";
    return `<span class="host-metric-number">${formatNumber(bytesToGiB(summaryBytes))}</span> <span class="host-metric-unit">GiB</span>`;
  }

  function formatHostMemoryPercentHTML(summaryBytes, memory) {
    if (!memory.hasMemory || memory.totalBytes <= 0) return "n/a";
    const percent = (summaryBytes / memory.totalBytes) * 100;
    return `${formatHostMemoryHTML(summaryBytes, true)} <span class="host-metric-percent">(${formatNumber(percent)}%)</span>`;
  }

  function formatHostPercentHTML(value) {
    if (value === null) return "n/a";
    return `<span class="host-metric-number">${formatNumber(value)}</span><span class="host-metric-unit">%</span>`;
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

  function bytesToMiB(value) {
    return Number(value || 0) / 1024 / 1024;
  }

  function numberOrNull(value) {
    return typeof value === "number" && Number.isFinite(value) ? value : null;
  }

  function textOrNA(value) {
    const text = String(value || "").trim();
    return text === "" ? "n/a" : text;
  }

  function deviceId(device) {
    if (device.index !== undefined && device.index !== null) return String(device.index);
    if (device.uuid) return String(device.uuid);
    return device.name || "unknown";
  }

  function deviceAliases(device, id) {
    const aliases = [];
    if (device.uuid && String(device.uuid) !== id) {
      aliases.push(String(device.uuid));
    }
    return aliases;
  }

  function deviceLabel(device) {
    const index = shortDeviceLabel(device);
    const name = deviceName(device);
    const uuid = device.uuid || "no UUID";
    return `${index} - ${name} - ${uuid}`;
  }

  function shortDeviceLabel(device) {
    return device.index !== undefined && device.index !== null ? `GPU ${device.index}` : "GPU";
  }

  function deviceName(device) {
    return device.name || "Unknown device";
  }

  function parseChartParam(value) {
    if (value === null || value === "") return null;
    if (value === "none") return [];
    if (value === "all") return chartOptions.map((option) => option.id);
    const parts = value.includes(",") ? value.split(",") : chartOptionMap.has(value) ? [value] : value.split("");
    const ids = parts.map(parseChartIDParam).filter(Boolean);
    return [...new Set(ids)];
  }

  function parseChartIDParam(value) {
    if (!value) return null;
    if (chartOptionMap.has(value)) return value;
    const index = Number(value);
    if (Number.isInteger(index) && index >= 0 && index < chartOptions.length) {
      return chartOptions[index].id;
    }
    return null;
  }

  function chartIDToParam(id) {
    return chartParamMap.get(id) || id;
  }

  function parseGPUParam(value) {
    if (value === null || value === "" || value === "none") return [];
    return value.split(",").filter(Boolean);
  }

  function parseHostConfig(value) {
    try {
      const hosts = JSON.parse(value || "[]");
      if (!Array.isArray(hosts)) return [];
      return hosts
        .map((host) => ({
          index: Number(host.index),
          name: String(host.name || "").trim(),
          hostName: String(host.host_name || "").trim(),
          default: Boolean(host.default),
        }))
        .filter((host) => Number.isInteger(host.index) && host.index >= 0 && host.name !== "" && host.hostName !== "");
    } catch {
      return [];
    }
  }

  function defaultHostOption(hosts) {
    const host = hosts.find((option) => option.default) || hosts[0];
    return host ? host.index : 0;
  }

  function parseHostParam(value, hosts, fallback) {
    const index = Number(value);
    if (Number.isInteger(index) && hosts.some((host) => host.index === index)) {
      return index;
    }
    return fallback;
  }

  function selectedChartIds() {
    return chartOptions.map((option) => option.id).filter((id) => state.selectedChartIds.has(id));
  }

  function isDefaultChartSelection() {
    const selected = selectedChartIds();
    return selected.length === defaultChartIds.length && selected.every((id, index) => id === defaultChartIds[index]);
  }

  function setTimeWindowMinutes(minutes) {
    const nextMinutes = Math.max(1, Math.round(Number(minutes) || 1));
    state.timeWindow = nextMinutes * 60000;
    dom.timeMinutes.value = String(nextMinutes);
    renderAll();
  }

  function applyTimeInput() {
    setTimeWindowMinutes(dom.timeMinutes.value);
  }

  function setRefreshIntervalSeconds(seconds) {
    const nextSeconds = Math.max(0, Math.round(Number(seconds) || 0));
    state.refreshInterval = nextSeconds * 1000;
    dom.refreshInterval.value = String(nextSeconds);
    scheduleRefresh();
    updateStatus();
  }

  function applyRefreshInput() {
    setRefreshIntervalSeconds(dom.refreshInterval.value);
  }

  function toggleDropdown(dropdown) {
    const isOpen = dropdown.classList.contains("is-open");
    closeDropdowns();
    if (!isOpen) {
      openDropdown(dropdown);
    }
  }

  function openDropdown(dropdown, closeOthers = true) {
    if (closeOthers) closeDropdowns();
    dropdown.classList.add("is-open");
    const trigger = dropdown.querySelector(".dropdown-trigger");
    if (trigger) trigger.setAttribute("aria-expanded", "true");
  }

  function focusTimeInput() {
    openDropdown(dom.timeDropdown);
    dom.timeMinutes.focus();
    dom.timeMinutes.select();
  }

  function focusRefreshInput() {
    openDropdown(dom.refreshDropdown);
    dom.refreshInterval.focus();
    dom.refreshInterval.select();
  }

  function closeDropdowns() {
    for (const dropdown of dom.dropdowns) {
      dropdown.classList.remove("is-open");
      const trigger = dropdown.querySelector(".dropdown-trigger");
      if (trigger) trigger.setAttribute("aria-expanded", "false");
    }
  }

  function updateStatus(kind, text) {
    const statusKind = kind || (state.refreshInterval > 0 ? "ok" : "warn");
    const statusText = text || statusLabelText();
    dom.statusDot.className = `status-dot is-${statusKind}`;
    dom.statusDot.setAttribute("aria-label", statusText);
    dom.statusTooltip.textContent = statusText;
  }

  function updateStatusTooltipPosition(event) {
    const margin = 10;
    const offset = 16;
    const width = dom.statusTooltip.offsetWidth || 160;
    const height = dom.statusTooltip.offsetHeight || 32;
    const x = Math.min(Math.max(event.clientX, width / 2 + margin), window.innerWidth - width / 2 - margin);
    let y = event.clientY + offset;
    if (y + height + margin > window.innerHeight) {
      y = event.clientY - height - offset;
    }
    dom.statusTooltip.style.setProperty("--status-tooltip-x", `${x}px`);
    dom.statusTooltip.style.setProperty("--status-tooltip-y", `${y}px`);
  }

  function statusLabelText() {
    if (state.refreshInterval <= 0) {
      return state.lastUpdate ? `Paused, ${formatTime(state.lastUpdate)}` : "Paused";
    }
    return state.lastUpdate ? `Updated ${formatTime(state.lastUpdate)}` : "Starting";
  }

  function escapeHTML(value) {
    return String(value).replace(/[&<>"']/g, (char) => ({
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    })[char]);
  }
})();
