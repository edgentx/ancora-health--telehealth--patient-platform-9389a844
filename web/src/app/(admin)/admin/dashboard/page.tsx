/** Clinic admin landing route (`/admin/dashboard`). */
export default function AdminDashboard() {
  return (
    <section>
      <h1 className="page-heading">Clinic overview</h1>
      <p className="page-subheading">Utilization, revenue, and compliance at a glance.</p>
      <div className="card-grid">
        <article className="card">
          <p className="card__label">Utilization</p>
          <p className="card__value">82%</p>
        </article>
        <article className="card">
          <p className="card__label">No-show rate</p>
          <p className="card__value">4.1%</p>
        </article>
        <article className="card">
          <p className="card__label">Revenue (MTD)</p>
          <p className="card__value">$248k</p>
        </article>
        <article className="card">
          <p className="card__label">Active providers</p>
          <p className="card__value">31</p>
        </article>
      </div>
    </section>
  );
}
