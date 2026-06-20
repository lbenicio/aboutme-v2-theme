/**
 * Blog search with tag/category filtering, pagination, URL sync.
 * Reads /blog/index.json for post data, renders entirely client-side.
 */
/* global URLSearchParams, clearTimeout, setTimeout */
(async function () {
  const script = document.currentScript;
  const INDEX_URL = script?.dataset?.indexUrl || "/post/index.json";
  const BASE_PATH = script?.dataset?.basePath || "";
  const CONTAINER_ID = "blog-results";
  const ITEMS_PER_PAGE = 12;

  // ---- State ----
  let allPosts = [];
  let activeTags = new Set();
  let activeCategories = new Set();
  let searchQuery = "";
  let currentPage = 1;

  // ---- DOM refs ----
  const searchInput = document.getElementById("blog-search-input");
  const tagsContainer = document.getElementById("blog-search-tags");
  const statusEl = document.getElementById("blog-search-status");
  const serverList = document.getElementById("blog-server-list") || document.getElementById("reading-server-list");

  // ---- Fetch index ----
  async function fetchPosts() {
    if (allPosts.length) return allPosts;
    try {
      const resp = await fetch(INDEX_URL);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      allPosts = await resp.json();
    } catch (e) {
      console.warn("Blog search: failed to load index, keeping server list", e);
      return [];
    }
    return allPosts;
  }

  // ---- URL sync ----
  function readURLParams() {
    try {
      const raw = window.location.hash.replace(/^#/, "");
      const p = new URLSearchParams(raw);
      const t = p.get("tags"),
        c = p.get("categories");
      activeTags = t ? new Set(t.split(",").filter(Boolean)) : new Set();
      activeCategories = c ? new Set(c.split(",").filter(Boolean)) : new Set();
      searchQuery = p.get("q") || "";
      currentPage = parseInt(p.get("page")) || 1;
      if (searchInput && document.activeElement !== searchInput) searchInput.value = searchQuery;
    } catch {
      // silence error
    }
  }

  function writeURLParams() {
    const sp = new URLSearchParams();
    if (activeTags.size) sp.set("tags", [...activeTags].sort().join(","));
    if (activeCategories.size) sp.set("categories", [...activeCategories].sort().join(","));
    if (searchQuery) sp.set("q", searchQuery);
    if (currentPage > 1) sp.set("page", String(currentPage));
    const qs = sp.toString();
    window.location.hash = qs ? "#" + qs : "";
  }

  // ---- Filter ----
  function filterPosts(posts) {
    return posts.filter((post) => {
      // Tag filter
      if (activeTags.size > 0) {
        const postTags = post.tags || [];
        if (![...activeTags].some((t) => postTags.includes(t))) return false;
      }
      // Category filter
      if (activeCategories.size > 0) {
        const postCats = post.categories || [];
        if (![...activeCategories].some((c) => postCats.includes(c))) return false;
      }
      // Text search
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        const haystack = [post.title || "", post.description || "", ...(post.tags || []), ...(post.categories || [])].join(" ").toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  }

  // ---- Collect all unique tags/categories with counts ----
  function collectFilters(posts) {
    const tagMap = new Map();
    const catMap = new Map();
    for (const p of posts) {
      (p.tags || []).forEach((t) => tagMap.set(t, (tagMap.get(t) || 0) + 1));
      (p.categories || []).forEach((c) => catMap.set(c, (catMap.get(c) || 0) + 1));
    }
    // Sort by count descending, then alphabetically
    const sortFn = (a, b) => b[1] - a[1] || a[0].localeCompare(b[0]);
    const tags = [...tagMap.entries()].sort(sortFn).map(([name, count]) => ({ type: "tag", value: name, count }));
    const categories = [...catMap.entries()].sort(sortFn).map(([name, count]) => ({ type: "category", value: name, count }));
    return { tags, categories };
  }

  // ---- Render tag pills ----
  function renderTagPills(filterData) {
    if (!tagsContainer) return;

    // Show the container (hidden on server) and apply flex layout
    tagsContainer.classList.remove("hidden");
    tagsContainer.classList.add("flex", "flex-wrap", "gap-2", "text-xs");

    const selected = new Set([...activeTags, ...activeCategories]);
    // Combine tags and categories, sorted by count (most popular first), deduplicated by value
    const seen = new Set();
    const all = [];
    for (const item of [...filterData.tags, ...filterData.categories].sort((a, b) => (b.count || 0) - (a.count || 0))) {
      if (!seen.has(item.value)) {
        seen.add(item.value);
        all.push(item);
      }
    }
    const BATCH_SIZE = 30;
    let visibleLimit = parseInt(tagsContainer.dataset.visibleCount) || BATCH_SIZE;
    if (visibleLimit < BATCH_SIZE) visibleLimit = BATCH_SIZE;

    let html = "";
    const renderItem = (item, isActive) => {
      const prefix = item.type === "tag" ? "#" : "";
      const activeClass = isActive ? "bg-accent text-accent-foreground ring-2 ring-accent" : "bg-muted text-muted-foreground border hover:bg-accent hover:text-accent-foreground";
      return `<button class="inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium transition-colors ${activeClass}" data-filter="${item.type}" data-value="${item.value}">${prefix}${item.value}${isActive ? " ×" : ""}</button>`;
    };

    // Always show active tags first, then popular tags
    const activeItems = all.filter((item) => selected.has(item.value));
    const inactiveItems = all.filter((item) => !selected.has(item.value));

    for (const item of activeItems) {
      html += renderItem(item, true);
    }

    let visibleCount = 0;
    for (const item of inactiveItems) {
      if (visibleCount < visibleLimit) {
        html += renderItem(item, false);
      }
      visibleCount++;
    }

    // Show more / show less toggle
    if (all.length > BATCH_SIZE) {
      const allVisible = visibleLimit >= all.length;
      const remaining = all.length - visibleLimit;
      html += `<button class="inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium bg-muted text-muted-foreground hover:bg-accent transition-colors" id="blog-toggle-tags">${allVisible ? "Show less" : `+${Math.min(BATCH_SIZE, remaining)} more`}</button>`;
    }

    // Clear all button — always visible
    const disabledAttr = selected.size === 0 ? "disabled" : "";
    const clearClass = selected.size > 0 ? "bg-destructive text-destructive-foreground hover:opacity-80" : "bg-muted text-muted-foreground opacity-50 cursor-not-allowed";
    html += `<button class="inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs font-medium transition-opacity ${clearClass}" id="blog-clear-filters" ${disabledAttr}>Clear all</button>`;

    tagsContainer.innerHTML = html;

    // Bind click events for tag/category toggles
    tagsContainer.querySelectorAll("button[data-filter]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const type = btn.dataset.filter;
        const value = btn.dataset.value;
        const set = type === "tag" ? activeTags : activeCategories;
        if (set.has(value)) {
          set.delete(value);
        } else {
          set.add(value);
        }
        currentPage = 1;
        writeURLParams();
        render();
      });
    });

    // Bind clear all
    const clearBtn = document.getElementById("blog-clear-filters");
    if (clearBtn) {
      clearBtn.addEventListener("click", () => {
        activeTags.clear();
        activeCategories.clear();
        searchQuery = "";
        if (searchInput) searchInput.value = "";
        currentPage = 1;
        writeURLParams();
        render();
      });
    }

    // Bind show more / show less toggle
    const toggleBtn = document.getElementById("blog-toggle-tags");
    if (toggleBtn) {
      toggleBtn.addEventListener("click", () => {
        const current = parseInt(tagsContainer.dataset.visibleCount) || 30;
        if (current >= all.length) {
          tagsContainer.dataset.visibleCount = 30;
        } else {
          tagsContainer.dataset.visibleCount = current + 30;
        }
        renderTagPills(collectFilters(allPosts));
      });
    }
  }

  // ---- Render posts ----
  function renderPosts(filtered) {
    // Hide server-rendered list and pagination
    if (serverList) serverList.style.display = "none";
    // Hide server pagination when client list is active
    const serverPagination = serverList?.nextElementSibling;
    if (serverPagination && serverPagination.tagName === "NAV") {
      serverPagination.style.display = "none";
    }

    let container = document.getElementById(CONTAINER_ID);
    if (!container) {
      container = document.createElement("ul");
      container.id = CONTAINER_ID;
      container.className = "grid gap-6 sm:grid-cols-2 lg:grid-cols-3";
      if (serverList) {
        serverList.parentNode.insertBefore(container, serverList.nextSibling);
      } else {
        const section = document.querySelector("#blog-search")?.closest("section");
        if (section) section.appendChild(container);
      }
    }

    if (filtered.length === 0) {
      container.innerHTML = `<li class="col-span-full text-sm text-muted-foreground py-8 text-center">No posts match your filters.</li>`;
      return;
    }

    const start = (currentPage - 1) * ITEMS_PER_PAGE;
    const page = filtered.slice(start, start + ITEMS_PER_PAGE);

    container.innerHTML = page
      .map(
        (post) => `
      <li class="border rounded-lg overflow-hidden flex flex-col transition-colors">
        ${post.cover ? `<a href="${post.link}" class="block aspect-video bg-muted"><img src="${post.cover}" alt="" class="w-full h-full object-cover" loading="lazy" /></a>` : ""}
        <div class="p-4 flex flex-col gap-2 grow">
          <h2 class="font-semibold text-lg leading-tight">
            <a href="${post.link}" class="underline underline-offset-4 hover:bg-muted">${post.title}</a>
          </h2>
          <div class="text-xs text-muted-foreground">${post.date || post.year || ""}</div>
          ${post.description ? `<p class="text-sm text-muted-foreground line-clamp-3">${post.description}</p>` : ""}
          ${
            post.tags?.length || post.categories?.length
              ? `
            <div class="mt-auto flex flex-wrap gap-2 text-xs text-muted-foreground">
              ${(post.categories || []).map((c) => `<a href="${BASE_PATH}/categories/${encodeURIComponent(c)}/" class="px-2 py-0.5 rounded bg-muted hover:bg-accent transition-colors">${c}</a>`).join("")}
              ${(post.tags || []).map((t) => `<a href="${BASE_PATH}/tags/${encodeURIComponent(t)}/" class="px-2 py-0.5 rounded bg-muted hover:bg-accent transition-colors">#${t}</a>`).join("")}
            </div>
          `
              : ""
          }
        </div>
      </li>`,
      )
      .join("");
  }

  // ---- Render pagination ----
  function renderPagination(total) {
    const totalPages = Math.ceil(total / ITEMS_PER_PAGE);

    // Remove old client pagination
    document.querySelector(".blog-client-pagination")?.remove();

    if (totalPages <= 1) return;

    const nav = document.createElement("nav");
    nav.className = "blog-client-pagination flex flex-col sm:flex-row items-center justify-between gap-4 pt-6";
    nav.setAttribute("role", "navigation");
    nav.setAttribute("aria-label", "Pagination");

    // Build page number buttons
    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);

    let pageButtons = "";
    if (startPage > 1) {
      pageButtons += `<button class="text-sm px-2 py-1.5 rounded-md hover:bg-muted transition-colors text-muted-foreground" data-page="1">1</button>`;
      if (startPage > 2) pageButtons += `<span class="text-sm px-1 text-muted-foreground">…</span>`;
    }
    for (let i = startPage; i <= endPage; i++) {
      if (i === currentPage) {
        pageButtons += `<span class="text-sm px-2 py-1.5 rounded-md bg-primary text-primary-foreground font-medium" aria-current="page">${i}</span>`;
      } else {
        pageButtons += `<button class="text-sm px-2 py-1.5 rounded-md hover:bg-muted transition-colors text-muted-foreground" data-page="${i}">${i}</button>`;
      }
    }
    if (endPage < totalPages) {
      if (endPage < totalPages - 1) pageButtons += `<span class="text-sm px-1 text-muted-foreground">…</span>`;
      pageButtons += `<button class="text-sm px-2 py-1.5 rounded-md hover:bg-muted transition-colors text-muted-foreground" data-page="${totalPages}">${totalPages}</button>`;
    }

    nav.innerHTML = `
      <span class="text-sm text-muted-foreground order-1">Page ${currentPage} of ${totalPages}</span>
      <div class="flex items-center gap-1 order-2 sm:order-3">
        ${currentPage > 1 ? `<button class="text-sm px-3 py-1.5 rounded-md border bg-background hover:bg-muted transition-colors" data-page="${currentPage - 1}" aria-label="Previous page">← Prev</button>` : ""}
        ${pageButtons}
        ${currentPage < totalPages ? `<button class="text-sm px-3 py-1.5 rounded-md border bg-background hover:bg-muted transition-colors" data-page="${currentPage + 1}" aria-label="Next page">Next →</button>` : ""}
      </div>
    `;

    nav.querySelectorAll("button[data-page]").forEach((btn) => {
      btn.addEventListener("click", () => {
        currentPage = parseInt(btn.dataset.page);
        render();
        window.scrollTo({ top: 0, behavior: "smooth" });
      });
    });

    const container = document.getElementById(CONTAINER_ID);
    if (container) container.parentNode.insertBefore(nav, container.nextSibling);
  }

  // ---- Main render ----
  function render() {
    const posts = allPosts.length ? allPosts : [];
    if (!posts.length) return;

    const filtered = filterPosts(posts);
    const filterData = collectFilters(posts);

    const hasFilters = activeTags.size > 0 || activeCategories.size > 0 || searchQuery;
    if (statusEl) {
      const total = filtered.length;
      statusEl.textContent = hasFilters ? `${total} post${total !== 1 ? "s" : ""} found` : "";
    }

    renderTagPills(filterData);

    if (!hasFilters && currentPage === 1) {
      const container = document.getElementById(CONTAINER_ID);
      if (container) container.innerHTML = "";
      document.querySelector(".blog-client-pagination")?.remove();
      if (serverList) serverList.style.display = "";
      const serverNav = serverList?.nextElementSibling;
      if (serverNav && serverNav.tagName === "NAV") serverNav.style.display = "";
      writeURLParams();
      return;
    }

    renderPosts(filtered);
    renderPagination(filtered.length);
  }

  // ---- Events ----
  if (searchInput) {
    let debounceTimer;
    searchInput.addEventListener("input", () => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        searchQuery = searchInput.value.trim();
        currentPage = 1;
        writeURLParams();
        render();
      }, 250);
    });
  }

  // Keyboard shortcut: press / to focus search
  document.addEventListener("keydown", (e) => {
    if (e.key === "/" && document.activeElement !== searchInput && document.activeElement?.tagName !== "INPUT") {
      e.preventDefault();
      searchInput?.focus();
    }
  });

  // ---- Init ----
  const posts = await fetchPosts();
  if (posts.length) {
    // Trigger render via hash (handles both initial load and hash changes)
    readURLParams();
    render();
  }

  // Listen for popstate (back/forward navigation)
  window.addEventListener("hashchange", () => {
    readURLParams();
    render();
  });
})();
