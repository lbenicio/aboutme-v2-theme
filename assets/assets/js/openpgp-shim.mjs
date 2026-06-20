// openpgp-shim.mjs
// Lightweight shim to load a local vendored OpenPGP bundle if present
// or fall back to the CDN. Exposes `openpgp` namespace as an ES module export.

export async function ensureOpenPGP() {
  // If global already present, return it
  if (typeof globalThis.openpgp !== "undefined") return globalThis.openpgp;

  // Try to load local vendored build first
  try {
    // The vendored file may be placed at assets/js/vendor/openpgp.min.mjs
    // which Hugo will copy to /assets/js/vendor/openpgp.min.mjs in public.
    const localPath = "/assets/js/vendor/openpgp.min.mjs";
    // Attempt to import the local module (if available at runtime)
    try {
      const mod = await import(localPath);
      if (mod && (mod.openpgp || globalThis.openpgp)) {
        return mod.openpgp || globalThis.openpgp;
      }
    } catch {
      // ignore local import failures and fall back to CDN
    }

    // Fallback to CDN UMD build loaded as script and exposed as global openpgp
    return await new Promise((resolve, reject) => {
      if (typeof document === "undefined") return reject(new Error("No document available"));
      const existing = document.querySelector("script[data-openpgp-shim]");
      if (existing) {
        existing.addEventListener("load", () => resolve(globalThis.openpgp));
        existing.addEventListener("error", () => reject(new Error("Failed to load OpenPGP script")));
        return;
      }
      const s = document.createElement("script");
      s.setAttribute("data-openpgp-shim", "1");
      s.src = "/assets/js/vendor/openpgp.umd.min.js"; // expected vendored UMD build path
      s.async = true;
      s.onload = () => {
        if (globalThis.openpgp) resolve(globalThis.openpgp);
        else reject(new Error("OpenPGP did not expose global"));
      };
      s.onerror = () => reject(new Error("Failed to load OpenPGP script"));
      document.head.appendChild(s);
    });
  } catch {
    // Last resort: dynamic import from unpkg CDN (ES module)
    try {
      const mod = await import("https://unpkg.com/openpgp@5.8.0/dist/openpgp.min.mjs");
      if (mod && (mod.openpgp || mod.default)) return mod.openpgp || mod.default || globalThis.openpgp;
    } catch {
      throw new Error("Unable to load OpenPGP library");
    }
  }
}
