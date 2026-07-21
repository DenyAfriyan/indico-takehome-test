import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type Stock = {
  item_id: string;
  total_stock: number;
  reserved_stock: number;
  available_stock: number;
};

type Reservation = {
  reservation_id: string;
  item_id: string;
  quantity: number;
  expires_at: string;
};

type APIError = {
  status?: string;
  error?: { code?: string; message?: string };
};

async function apiRequest<T>(path: string, options?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...options,
      headers: { "Content-Type": "application/json", ...options?.headers },
    });
  } catch {
    throw new Error("Cannot reach the inventory API. Make sure the Go backend is running on port 8080.");
  }

  const payload = (await response.json().catch(() => ({}))) as T & APIError;
  if (!response.ok) {
    throw new Error(payload.error?.message ?? `Request failed with status ${response.status}.`);
  }
  return payload;
}

function formatCountdown(milliseconds: number) {
  const safeSeconds = Math.max(0, Math.ceil(milliseconds / 1000));
  const minutes = Math.floor(safeSeconds / 60);
  const seconds = safeSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export default function App() {
  const [itemID, setItemID] = useState("item_4021");
  const [trackedItem, setTrackedItem] = useState("item_4021");
  const [userID, setUserID] = useState("usr_9981");
  const [quantity, setQuantity] = useState("1");
  const [stock, setStock] = useState<Stock | null>(null);
  const [reservation, setReservation] = useState<Reservation | null>(null);
  const [remaining, setRemaining] = useState(0);
  const [loadingStock, setLoadingStock] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const loadStock = useCallback(async (targetItem = trackedItem, quiet = false) => {
    if (!targetItem.trim()) return;
    if (!quiet) setLoadingStock(true);
    try {
      const result = await apiRequest<Stock>(
        `/api/v1/inventory/stock?item_id=${encodeURIComponent(targetItem.trim())}`,
      );
      setStock(result);
      setError("");
      setLastUpdated(new Date());
    } catch (requestError) {
      setStock(null);
      setError(requestError instanceof Error ? requestError.message : "Unable to load inventory.");
    } finally {
      if (!quiet) setLoadingStock(false);
    }
  }, [trackedItem]);

  useEffect(() => {
    void loadStock();
    const poll = window.setInterval(() => void loadStock(trackedItem, true), 5000);
    return () => window.clearInterval(poll);
  }, [loadStock, trackedItem]);

  useEffect(() => {
    if (!reservation) return;
    const update = () => {
      const timeLeft = new Date(reservation.expires_at).getTime() - Date.now();
      setRemaining(Math.max(0, timeLeft));
      if (timeLeft <= 0) {
        setReservation(null);
        setNotice("");
        setError("This reservation expired. Its stock has been returned to availability.");
        void loadStock(reservation.item_id, true);
      }
    };
    update();
    const timer = window.setInterval(update, 1000);
    return () => window.clearInterval(timer);
  }, [reservation, loadStock]);

  const availabilityPercent = useMemo(() => {
    if (!stock || stock.total_stock === 0) return 0;
    return Math.round((stock.available_stock / stock.total_stock) * 100);
  }, [stock]);

  const quantityValue = Number(quantity);
  const quantityIsValid = Number.isInteger(quantityValue) && quantityValue > 0;

  function handleTrackItem(event: FormEvent) {
    event.preventDefault();
    const nextItem = itemID.trim();
    if (!nextItem) {
      setError("Enter an item ID to track.");
      return;
    }
    setTrackedItem(nextItem);
    setReservation(null);
    setNotice("");
  }

  async function handleReserve(event: FormEvent) {
    event.preventDefault();
	if (!quantityIsValid) {
		setError("Quantity must be a whole number greater than zero.");
		return;
	}
    setError("");
    setNotice("");
    setSubmitting(true);
    try {
      const result = await apiRequest<Reservation>("/api/v1/inventory/reserve", {
        method: "POST",
        body: JSON.stringify({ user_id: userID.trim(), item_id: trackedItem, quantity: quantityValue }),
      });
      setReservation(result);
	  setNotice(`${quantityValue} unit${quantityValue === 1 ? "" : "s"} held successfully.`);
      await loadStock(trackedItem, true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Reservation failed.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleConfirm() {
    if (!reservation) return;
    setError("");
    setConfirming(true);
    try {
      await apiRequest("/api/v1/inventory/confirm", {
        method: "POST",
        body: JSON.stringify({ reservation_id: reservation.reservation_id }),
      });
      const confirmedQuantity = reservation.quantity;
      setReservation(null);
      setNotice(`Purchase confirmed for ${confirmedQuantity} unit${confirmedQuantity === 1 ? "" : "s"}.`);
      await loadStock(trackedItem, true);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Confirmation failed.");
    } finally {
      setConfirming(false);
    }
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <a className="brand" href="#top" aria-label="Stockroom dashboard home">
          <span className="brand-mark" aria-hidden="true"><span /></span>
          <span>Stockroom</span>
        </a>
        <div className="environment"><span className="status-dot" /> API environment <b>Local</b></div>
      </header>

      <div className="dashboard" id="top">
        <section className="hero-row">
          <div>
            <p className="eyebrow">Inventory operations</p>
            <h1>Reservation control room</h1>
            <p className="subtitle">Monitor live availability, hold inventory, and confirm purchases before time runs out.</p>
          </div>
          <form className="item-search" onSubmit={handleTrackItem}>
            <label htmlFor="item-id">Tracking item</label>
            <div className="search-control">
              <input id="item-id" value={itemID} onChange={(event) => setItemID(event.target.value)} placeholder="item_4021" />
              <button type="submit">Load item</button>
            </div>
          </form>
        </section>

        <section className="metrics" aria-label="Inventory summary">
          <article className="metric-card total">
            <div className="metric-label"><span className="metric-icon">Σ</span> Total stock</div>
            <strong>{loadingStock ? "—" : stock?.total_stock ?? "—"}</strong>
            <span>Physical units in inventory</span>
          </article>
          <article className="metric-card reserved">
            <div className="metric-label"><span className="metric-icon">◷</span> Reserved</div>
            <strong>{loadingStock ? "—" : stock?.reserved_stock ?? "—"}</strong>
            <span>Units currently on hold</span>
          </article>
          <article className="metric-card available">
            <div className="metric-label"><span className="metric-icon">✓</span> Available now</div>
            <strong>{loadingStock ? "—" : stock?.available_stock ?? "—"}</strong>
            <span>{stock ? `${availabilityPercent}% of physical stock` : "Waiting for inventory data"}</span>
            <div className="availability-track"><span style={{ width: `${availabilityPercent}%` }} /></div>
          </article>
        </section>

        <div className="feedback-region" aria-live="polite">
          {error && <div className="alert error-alert"><span aria-hidden="true">!</span><div><b>Request unsuccessful</b><p>{error}</p></div><button onClick={() => setError("")} aria-label="Dismiss error">×</button></div>}
          {notice && !error && <div className="alert success-alert"><span aria-hidden="true">✓</span><div><b>All set</b><p>{notice}</p></div><button onClick={() => setNotice("")} aria-label="Dismiss message">×</button></div>}
        </div>

        <section className="workspace-grid">
          <article className="panel reserve-panel">
            <div className="panel-heading">
              <div><p className="step-label">Step 01</p><h2>Create a reservation</h2></div>
              <span className="panel-badge">5 min hold</span>
            </div>
            <p className="panel-copy">Temporarily hold stock for a customer. Unconfirmed units return automatically when the timer ends.</p>
            <form className="reservation-form" onSubmit={handleReserve}>
              <label htmlFor="user-id">User ID</label>
              <input id="user-id" value={userID} onChange={(event) => setUserID(event.target.value)} placeholder="usr_9981" required />
              <div className="quantity-row">
				<div><label htmlFor="quantity">Quantity</label><div className="number-input"><button type="button" onClick={() => setQuantity(String(Math.max(1, (Number(quantity) || 1) - 1)))} aria-label="Decrease quantity">−</button><input id="quantity" type="number" min="1" step="1" inputMode="numeric" value={quantity} onChange={(event) => setQuantity(event.target.value)} /><button type="button" onClick={() => setQuantity(String((Number(quantity) || 0) + 1))} aria-label="Increase quantity">+</button></div></div>
                <div className="item-context"><span>Selected item</span><b>{trackedItem}</b></div>
              </div>
			  <button className="primary-button" disabled={submitting || !!reservation || !stock || !quantityIsValid || stock.available_stock < quantityValue}>
				{submitting ? "Reserving…" : reservation ? "Reservation already active" : "Reserve inventory"}<span aria-hidden="true">→</span>
              </button>
			  {!quantityIsValid && quantity !== "" && <p className="form-hint danger">Enter a whole number greater than zero.</p>}
			  {stock && quantityIsValid && stock.available_stock < quantityValue && <p className="form-hint danger">Quantity exceeds available stock.</p>}
			  {reservation && <p className="form-hint">Confirm or wait for the active reservation to expire before reserving again.</p>}
            </form>
          </article>

          <article className={`panel confirmation-panel ${reservation ? "is-active" : ""}`}>
            <div className="panel-heading">
              <div><p className="step-label">Step 02</p><h2>Confirm purchase</h2></div>
              {reservation && <span className="live-badge"><span /> Active</span>}
            </div>
            {!reservation ? (
              <div className="empty-state">
                <div className="empty-visual"><span>00</span><i>:</i><span>00</span></div>
                <h3>No active reservation</h3>
                <p>Create a reservation and its confirmation timer will appear here.</p>
              </div>
            ) : (
              <div className="active-reservation">
                <div className="countdown-block">
                  <span>Time remaining</span>
                  <strong>{formatCountdown(remaining)}</strong>
                  <div className="timer-track"><span style={{ width: `${Math.min(100, (remaining / 300000) * 100)}%` }} /></div>
                </div>
                <dl className="reservation-details">
                  <div><dt>Reservation</dt><dd>{reservation.reservation_id}</dd></div>
                  <div><dt>Item</dt><dd>{reservation.item_id}</dd></div>
                  <div><dt>Quantity</dt><dd>{reservation.quantity} units</dd></div>
                  <div><dt>Expires</dt><dd>{new Date(reservation.expires_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}</dd></div>
                </dl>
                <button className="confirm-button" onClick={handleConfirm} disabled={confirming || remaining <= 0}>{confirming ? "Confirming…" : "Confirm purchase"}<span aria-hidden="true">✓</span></button>
                <p className="secure-note"><span aria-hidden="true">◆</span> Confirmation permanently commits this inventory.</p>
              </div>
            )}
          </article>
        </section>

        <footer className="dashboard-footer">
          <span><i className={`connection-dot ${stock ? "connected" : ""}`} /> {stock ? "Backend connected" : "Backend unavailable"}</span>
          <span>{lastUpdated ? `Updated ${lastUpdated.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}` : "Waiting for first update"} · Auto-refresh every 5s</span>
          <button onClick={() => void loadStock()} disabled={loadingStock}>{loadingStock ? "Refreshing…" : "Refresh now"}</button>
        </footer>
      </div>
    </main>
  );
}
