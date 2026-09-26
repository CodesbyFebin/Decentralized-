import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Empty } from '../common/states';

export default function NotFoundView() {
  const loc = useLocation();
  return (
    <div className="pt-6 max-w-xl">
      <Empty title="Page not found" detail={`Nothing lives at ${loc.pathname}.`} action={<Link to="/" className="text-cyan-300 hover:underline">Back to the dashboard</Link>} />
    </div>
  );
}
