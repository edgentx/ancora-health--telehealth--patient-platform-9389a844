/** Provider landing route (`/provider/dashboard`). */
export default function ProviderDashboard() {
  return (
    <section>
      <h1 className="page-heading">Today at a glance</h1>
      <p className="page-subheading">Your schedule, pending notes, and patient messages.</p>
      <div className="card-grid">
        <article className="card">
          <p className="card__label">Visits today</p>
          <p className="card__value">8</p>
        </article>
        <article className="card">
          <p className="card__label">Notes to sign</p>
          <p className="card__value">2</p>
        </article>
        <article className="card">
          <p className="card__label">New messages</p>
          <p className="card__value">5</p>
        </article>
        <article className="card">
          <p className="card__label">Refill requests</p>
          <p className="card__value">1</p>
        </article>
      </div>
    </section>
  );
}
