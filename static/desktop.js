let currentZIndex = 10;

document.addEventListener("DOMContentLoaded", () => {
  initializeWindows();
  initializeDesktopIcons();
  initializeGuestbook();
  initializeExplorer();
});

window.addEventListener("resize", () => {
  const windows = document.querySelectorAll(".window");
  windows.forEach((win) => {
    if (win.dataset.maximized === "true") {
      setWindowSize(win);
    }
  });
});

// ============================================
// HELPERS
// ============================================

// Toggle window maximize/restore state
function toggleMaximize(win) {
  if (win.dataset.maximized === "true") {
    win.style.width = win.dataset.originalWidth;
    win.style.height = win.dataset.originalHeight;
    win.style.top = win.dataset.originalTop;
    win.style.left = win.dataset.originalLeft;
    win.dataset.maximized = "false";
  } else setWindowSize(win);
}

function setWindowSize(win) {
  if (win.dataset.maximized !== "true") {
    win.dataset.originalWidth = win.style.width || win.offsetWidth + "px";
    win.dataset.originalHeight = win.style.height || win.offsetHeight + "px";
    win.dataset.originalTop = win.style.top || "50px"; // Garante um fallback caso esteja vazio
    win.dataset.originalLeft = win.style.left || "50px";
  }

  win.style.height = window.innerHeight - 31 + "px";
  win.style.top = "0";
  win.style.left = "0";
  win.style.width = window.innerWidth + "px";
  win.dataset.maximized = "true";
}

// ============================================
// WINDOW INITIALIZATION
// ============================================
function initializeWindows() {
  const windows = document.querySelectorAll(".window");
  windows.forEach((win) => {
    attachDragBehavior(win);
    attachButtonBehavior(win);
    if (win.style.display !== "none") {
      addWindowToTaskbar(win);
    }
  });
}

// ============================================
// DRAG AND DROP BEHAVIOR
// ============================================
function attachDragBehavior(win) {
  const titleBar = win.querySelector(".title-bar");
  let isMouseDragging = false;

  titleBar.addEventListener("mousedown", (e) => {
    if (e.target.tagName === "BUTTON") return;

    bringToFront(win);
    win.classList.add("is-dragging");

    let shiftX = 0;
    let shiftY = 0;
    let hasRestoredThisDrag = false;

    if (win.dataset.maximized !== "true") {
      const rect = win.getBoundingClientRect();
      shiftX = e.clientX - rect.left;
      shiftY = e.clientY - rect.top;
    }

    // Move the window following the mouse
    const onMouseMove = (moveEvent) => {
      isMouseDragging = true;

      if (win.dataset.maximized === "true") {
        if (!hasRestoredThisDrag) {
          toggleMaximize(win);

          // Force synchronous DOM reflow to get the restored dimensions immediately
          void win.offsetWidth;

          let newLeft = moveEvent.clientX - win.offsetWidth / 2;
          let newTop = moveEvent.clientY - 20;

          // Apply boundary checks
          newLeft = Math.max(
            0,
            Math.min(newLeft, window.innerWidth - win.offsetWidth),
          );
          newTop = Math.max(
            0,
            Math.min(newTop, window.innerHeight - win.offsetHeight - 30),
          );

          // Apply the new position
          win.style.left = newLeft + "px";
          win.style.top = newTop + "px";

          // offset recalc
          shiftX = moveEvent.clientX - newLeft;
          shiftY = moveEvent.clientY - newTop;

          hasRestoredThisDrag = true;
        }
      }

      let newX = moveEvent.clientX - shiftX;
      let newY = moveEvent.clientY - shiftY;

      // Keep window within viewport boundaries
      newX = Math.max(0, Math.min(newX, window.innerWidth - win.offsetWidth));
      newY = Math.max(
        0,
        Math.min(newY, window.innerHeight - win.offsetHeight - 30),
      );

      win.style.left = newX + "px";
      win.style.top = newY + "px";
    };

    // Clean up event listeners when mouse is released
    const onMouseUp = () => {
      isMouseDragging = false;
      win.classList.remove("is-dragging");
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };

    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
  });

  // Maximize/restore on double click on the title bar
  titleBar.addEventListener("dblclick", (e) => {
    if (e.target.tagName === "BUTTON") return;
    toggleMaximize(win);
  });

  // Bring to front on any click within the window
  win.addEventListener("mousedown", () => bringToFront(win));
}

// ============================================
// BUTTON BEHAVIOR
// ============================================
function attachButtonBehavior(win) {
  const btnClose = win.querySelector(".btn-close");
  const btnMinimize = win.querySelector(".btn-minimize");
  const btnMaximize = win.querySelector(".btn-maximize");

  if (btnClose) {
    btnClose.addEventListener("click", () => {
      win.style.display = "none";
      removeWindowFromTaskbar(win.id);
    });
  }

  if (btnMinimize) {
    btnMinimize.addEventListener("click", () => {
      win.style.display = "none";
    });
  }

  if (btnMaximize) {
    btnMaximize.addEventListener("click", () => {
      toggleMaximize(win);
    });
  }
}

// ============================================
// Z-INDEX
// ============================================
function bringToFront(win) {
  currentZIndex++;
  win.style.zIndex = currentZIndex;

  if (win.style.display === "none") {
    win.style.display = "block";
    addWindowToTaskbar(win);
  }
}

// ============================================
// DESKTOP ICONS
// ============================================
function initializeDesktopIcons() {
  const icons = document.querySelectorAll(".desktop-icon");

  icons.forEach((icon) => {
    icon.addEventListener("dblclick", () => {
      const windowId = icon.dataset.window;
      const win = document.getElementById(windowId);

      if (win) {
        win.style.display = "block";
        bringToFront(win);
        addWindowToTaskbar(win);

        if (windowId === "win-guestbook") {
          loadGuestbookEntries();
        }
      }
    });
  });
}

// ============================================
// TASKBAR MANAGEMENT
// ============================================
function addWindowToTaskbar(win) {
  const taskbar = document.getElementById("taskbar-windows");
  const windowId = win.id;

  if (document.getElementById(`taskbar-${windowId}`)) return;

  const btn = document.createElement("button");
  btn.id = `taskbar-${windowId}`;
  btn.className = "taskbar-button";
  btn.textContent = win.querySelector(".title-bar span").textContent;

  btn.addEventListener("click", () => {
    if (win.style.display === "none") {
      win.style.display = "block";
      bringToFront(win);
      if (windowId === "win-guestbook") {
        loadGuestbookEntries();
      }
    } else {
      win.style.display = "none";
    }
  });

  taskbar.appendChild(btn);
}

function removeWindowFromTaskbar(windowId) {
  const btn = document.getElementById(`taskbar-${windowId}`);
  if (btn) btn.remove();
}

// ============================================
// GUESTBOOK
// ============================================
const LS_SEEN = "guestbook_seen";
const LS_WRITTEN = "guestbook_written";

function initializeGuestbook() {
  const hasSeen = localStorage.getItem(LS_SEEN);
  const hasWritten = localStorage.getItem(LS_WRITTEN);

  // cache DOM elements
  const targetDiv = document.getElementById("ico-guestbook");
  const winWelcome = document.getElementById("win-welcome");
  const writeSection = document.getElementById("guestbook-write-section");
  const welcomeNameInput = document.getElementById("welcome-name-input");
  const welcomeWriteBtn = document.getElementById("welcome-write-btn");
  const welcomeSkipBtn = document.getElementById("welcome-skip-btn");
  const guestbookNameInput = document.getElementById("guestbook-name-input");
  const guestbookSubmitBtn = document.getElementById("guestbook-submit-btn");
  const welcomeClose = document.querySelector("#win-welcome .btn-close");

  // helper to reveal the guestbook icon
  function showTargetDiv() {
    if (targetDiv) targetDiv.classList.remove("hidden");
  }

  // show welcome window only on first visit
  if (!hasSeen) {
    if (winWelcome) {
      winWelcome.style.display = "block";
      bringToFront(winWelcome);
    }
    // hide icon using classlist to respect css cascade
    if (targetDiv) targetDiv.classList.add("hidden");
  } else {
    showTargetDiv();
  }

  if (writeSection) {
    writeSection.style.display = hasWritten ? "none" : "block";
  }

  if (welcomeNameInput && welcomeWriteBtn) {
    const onWelcomeSubmit = () => {
      submitGuestbookEntry(welcomeNameInput.value, () => {
        localStorage.setItem(LS_SEEN, "1");
        localStorage.setItem(LS_WRITTEN, "1");
        if (winWelcome) winWelcome.style.display = "none";
        showTargetDiv();
        if (writeSection) writeSection.style.display = "none";
      });
    };

    welcomeWriteBtn.addEventListener("click", onWelcomeSubmit);

    welcomeNameInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault(); // prevent default form submission behavior
        onWelcomeSubmit();
      }
    });
  }

  if (welcomeSkipBtn) {
    welcomeSkipBtn.addEventListener("click", () => {
      localStorage.setItem(LS_SEEN, "1");
      if (winWelcome) winWelcome.style.display = "none";
      showTargetDiv();
    });
  }

  if (guestbookNameInput && guestbookSubmitBtn) {
    const onGuestbookSubmit = () => {
      submitGuestbookEntry(guestbookNameInput.value, () => {
        localStorage.setItem(LS_WRITTEN, "1");
        if (writeSection) writeSection.style.display = "none";
        guestbookNameInput.value = ""; // Clear input after success
      });
    };

    guestbookSubmitBtn.addEventListener("click", onGuestbookSubmit);

    guestbookNameInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        onGuestbookSubmit();
      }
    });
  }

  if (welcomeClose) {
    welcomeClose.addEventListener("click", () => {
      localStorage.setItem(LS_SEEN, "1");
      if (winWelcome) winWelcome.style.display = "none";
      showTargetDiv();
    });
  }
}

function submitGuestbookEntry(name, onSuccess) {
  name = name.trim();
  if (!name) {
    alert("plz enter a name!");
    return;
  }

  fetch("/api/guestbook", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: name }),
  })
    .then((res) => {
      if (!res.ok) throw new Error("Failed to save");
      return res.json();
    })
    .then(() => {
      onSuccess();
      loadGuestbookEntries();
    })
    .catch((err) => {
      console.error("Guestbook error:", err);
      alert("Error saving entry: " + err.message);
    });
}

function loadGuestbookEntries() {
  fetch("/api/guestbook")
    .then((res) => res.json())
    .then((entries) => {
      const list = document.getElementById("guestbook-list");
      if (!list) return;

      if (entries.length === 0) {
        list.innerHTML =
          "<p class='guestbook-empty'>no signatures yet. be the first!</p>";
        return;
      }

      list.innerHTML = entries
        .slice()
        .reverse()
        .map(
          (entry) =>
            `<div class="guestbook-entry">
                <span class="guestbook-name">${escapeHtml(entry.name)}</span>
                <span class="guestbook-date">${entry.timestamp}</span>
            </div>`,
        )
        .join("");
    })
    .catch((err) => console.error("failed to load entries:", err));
}

// ============================================
// FILE EXPLORER
// ============================================
let currentExplorerPath = "/";

// extensions the browser renders natively
const VIEWABLE_EXTENSIONS = [
  "png",
  "jpg",
  "jpeg",
  "gif",
  "webp",
  "svg",
  "txt",
  "md",
  "pdf",
  "mp4",
  "webm",
  "mp3",
  "wav",
  "ogg",
];

function initializeExplorer() {
  const explorerIcon = document.querySelector('[data-window="win-explorer"]');
  if (explorerIcon) {
    explorerIcon.addEventListener("dblclick", () => {
      const win = document.getElementById("win-explorer");
      if (win) {
        win.style.display = "block";
        bringToFront(win);
        addWindowToTaskbar(win);
        loadExplorerDirectory("/");
      }
    });
  }

  // navigateUp is a "cd ../" like function (has api protections too)
  const upBtn = document.getElementById("explorer-up-btn");
  if (upBtn) {
    upBtn.addEventListener("click", () => {
      navigateUp();
    });
  }
}

function loadExplorerDirectory(path) {
  currentExplorerPath = path;
  const listContainer = document.getElementById("explorer-list");
  const pathDisplay = document.getElementById("explorer-path-display");
  const upBtn = document.getElementById("explorer-up-btn");

  if (pathDisplay) pathDisplay.textContent = path === "/" ? "/" : path;
  if (upBtn) upBtn.disabled = path === "/";
  if (listContainer)
    listContainer.innerHTML = '<p class="explorer-loading">Loading...</p>';

  fetch(`/api/files?path=${encodeURIComponent(path)}`)
    .then((res) => {
      if (!res.ok) throw new Error("failed to load directory");
      return res.json();
    })
    .then((items) => {
      if (!listContainer) return;
      listContainer.innerHTML = "";

      if (items.length <= 1) {
        listContainer.innerHTML +=
          '<p class="explorer-empty">this folder is empty</p>';
        return;
      }

      items.forEach((item) => {
        const div = document.createElement("div");
        div.className = `explorer-item ${item.is_dir ? "is-dir" : "is-file"}`;

        const icon = item.is_dir ? "📁" : "📄";

        // condition so size isnt showed for directories
        if (item.is_dir !== false) {
          div.innerHTML = `
                    <span class="explorer-icon">${icon}</span>
                    <span class="explorer-name">${escapeHtml(item.name)}</span>
                `;
        } else {
          div.innerHTML = `
                    <span class="explorer-icon">${icon}</span>
                    <span class="explorer-name">${escapeHtml(item.name)}</span>
                    <span class="explorer-size">${formatFileSize(item.size)}</span>
                `;
        }

        div.addEventListener("dblclick", () => {
          if (item.name === "..") {
            navigateUp();
          } else if (item.is_dir) {
            const newPath =
              currentExplorerPath === "/"
                ? `/${item.name}`
                : `${currentExplorerPath}/${item.name}`;
            loadExplorerDirectory(newPath);
          } else {
            openPublicFile(item.name);
          }
        });

        listContainer.appendChild(div);
      });
    })
    .catch((err) => {
      console.error("Explorer error:", err);
      if (listContainer)
        listContainer.innerHTML =
          '<p class="explorer-error">error loading files</p>';
    });
}

function navigateUp() {
  if (currentExplorerPath === "/") return;
  const parts = currentExplorerPath.split("/").filter((p) => p !== "");
  parts.pop();
  const newPath = parts.length === 0 ? "/" : "/" + parts.join("/");
  loadExplorerDirectory(newPath);
}

function formatFileSize(bytes) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

// builds the public url for a file inside the current directory
function buildPublicUrl(name) {
  const base = currentExplorerPath === "/" ? "" : currentExplorerPath;
  return "/public" + base + "/" + encodeURIComponent(name);
}

// checks if the browser can render the file natively
function isViewableFile(name) {
  const parts = name.split(".");
  if (parts.length < 2) return false;
  const ext = parts.pop().toLowerCase();
  return VIEWABLE_EXTENSIONS.includes(ext);
}

// opens viewable files in a new tab, downloads everything else
function openPublicFile(name) {
  const url = buildPublicUrl(name);

  if (isViewableFile(name)) {
    window.open(url, "_blank");
    return;
  }

  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  link.remove();
}

// =============================================================
// etc
// =============================================================

// Prevents XSS attacks when rendering user input
function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}
