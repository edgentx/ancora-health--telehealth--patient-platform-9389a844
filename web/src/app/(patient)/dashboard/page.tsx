/** Patient landing route (`/dashboard`). */
export default function PatientDashboard() {
  return (
    <section>
      <h1 className="page-heading">Your health, at a glance</h1>
      <p className="page-subheading">Upcoming visits, messages, and care tasks.</p>
      <div className="card-grid">
        <article className="card">
          <p className="card__label">Upcoming appointments</p>
          <p className="card__value">2</p>
        </article>
        <article className="card">
          <p className="card__label">Unread messages</p>
          <p className="card__value">3</p>
        </article>
        <article className="card">
          <p className="card__label">Active prescriptions</p>
          <p className="card__value">4</p>
        </article>
        <article className="card">
          <p className="card__label">Balance due</p>
          <p className="card__value">$0.00</p>
        </article>
      </div>
    </section>
  );
}
