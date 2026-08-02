import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import DashboardLayout from '../components/DashboardLayout';
import type { RepoSummary } from '../types';

export default function RepoListPage() {
  const [repos, setRepos] = useState<RepoSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/v1/repos', { credentials: 'include' })
      .then(res => res.json())
      .then(data => { setRepos(data); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  return (
    <DashboardLayout>
      <h1 className="text-3xl font-bold mb-8">Repositories</h1>
      {loading ? <p>Loading...</p> : (
        <div className="grid gap-4">
          {repos.map(repo => (
            <Link to={`/dashboard/repos/${repo.id}`} key={repo.id} className="p-4 bg-white rounded shadow border hover:shadow-md transition-shadow">
              <h2 className="text-xl font-semibold">{repo.full_name}</h2>
              <div className="mt-2 flex gap-6 text-sm">
                <span className="text-red-600 font-medium">High: {repo.high_severity}</span>
                <span className="text-gray-600">Open: {repo.open_findings}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </DashboardLayout>
  );
}