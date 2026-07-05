/** Scheduler landing route (`/scheduler/dashboard`). */
export default function SchedulerDashboard() {
  return (
    <section>
      <h1 className="page-heading">Front desk</h1>
      <p className="page-subheading">Check-ins, the daily schedule, and eligibility tasks.</p>
      <div className="card-grid">
        <article className="card">
          <p className="card__label">Arrivals to check in</p>
          <p className="card__value">6</p>
        </article>
        <article className="card">
          <p className="card__label">Appointments today</p>
          <p className="card__value">27</p>
        </article>
        <article className="card">
          <p className="card__label">Eligibility to verify</p>
          <p className="card__value">4</p>
        </article>
        <article className="card">
          <p className="card__label">Open slots</p>
          <p className="card__value">9</p>
        </article>
      </div>
    </section>
  );
}
