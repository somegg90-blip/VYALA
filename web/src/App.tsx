import { useEffect, useState } from 'react';
import { Routes, Route, Link, useParams } from 'react-router-dom';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface RepoSummary {
  id: string;
  full_name: string;
  open_findings: number;
  high_severity: number;
  last_scanned_at: string;
}

interface TrendPoint {
  date: string;
  high: number;
  medium: number;
  low: number;
}

interface Finding {
  id: string;
  file: string;
  line: number;
  algorithm: string;
  severity: string;
  category: string;
  suggested_replacement: string;
}

function RepoList() {
  const [repos, setRepos] = useState<RepoSummary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/v1/repos')
      .then(res => res.json())
      .then(data => { setRepos(data); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  return (
    <div className="min-h-screen p-8 max-w-5xl mx-auto">
      <h1 className="text-3xl font-bold mb-8">VYALA Dashboard</h1>
      {loading ? <p>Loading...</p> : (
        <div className="grid gap-4">
          {repos.map(repo => (
            <Link to={`/repos/${repo.id}`} key={repo.id} className="p-4 bg-white rounded shadow border hover:shadow-md transition-shadow">
              <h2 className="text-xl font-semibold">{repo.full_name}</h2>
              <div className="mt-2 flex gap-6 text-sm">
                <span className="text-red-600 font-medium">High: {repo.high_severity}</span>
                <span className="text-gray-600">Open: {repo.open_findings}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function RepoDetail() {
  const { id } = useParams();
  const [trends, setTrends] = useState<TrendPoint[]>([]);
  const [findings, setFindings] = useState<Finding[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    Promise.all([
      fetch(`/v1/repos/${id}/trends`).then(res => res.json()),
      fetch(`/v1/repos/${id}/findings`).then(res => res.json())
    ]).then(([trendData, findingData]) => {
      setTrends(trendData);
      setFindings(findingData);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [id]);

  if (loading) return <div className="p-8">Loading repo details...</div>;

  return (
    <div className="min-h-screen p-8 max-w-6xl mx-auto">
      <Link to="/" className="text-blue-600 mb-4 inline-block">&larr; Back to Repositories</Link>
      <h1 className="text-3xl font-bold mb-8">Repository Posture</h1>

      {/* Trend Chart */}
      <div className="bg-white p-4 rounded shadow border mb-8">
        <h2 className="text-xl font-semibold mb-4">Open Findings (Last 90 Days)</h2>
        <div className="h-72 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={trends}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="high" stroke="#dc2626" strokeWidth={2} />
              <Line type="monotone" dataKey="medium" stroke="#f59e0b" strokeWidth={2} />
              <Line type="monotone" dataKey="low" stroke="#3b82f6" strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Findings Table */}
      <div className="bg-white p-4 rounded shadow border">
        <h2 className="text-xl font-semibold mb-4">Current Open Findings ({findings.length})</h2>
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead className="bg-gray-50 border-b">
              <tr>
                <th className="px-4 py-2 text-left">Severity</th>
                <th className="px-4 py-2 text-left">File</th>
                <th className="px-4 py-2 text-left">Algorithm</th>
                <th className="px-4 py-2 text-left">Category</th>
                <th className="px-4 py-2 text-left">Suggested Replacement</th>
              </tr>
            </thead>
            <tbody>
              {findings.map(f => (
                <tr key={f.id} className="border-b">
                  <td className="px-4 py-2 font-medium">
                    <span className={f.severity === 'high' ? 'text-red-600' : f.severity === 'medium' ? 'text-yellow-600' : 'text-blue-600'}>
                      {f.severity}
                    </span>
                  </td>
                  <td className="px-4 py-2 font-mono">{f.file}:{f.line}</td>
                  <td className="px-4 py-2">{f.algorithm}</td>
                  <td className="px-4 py-2">{f.category}</td>
                  <td className="px-4 py-2 text-gray-600">{f.suggested_replacement}</td>
                </tr>
              ))}
              {findings.length === 0 && (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No open vulnerabilities! Nice work.</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function App() {
  return (
    <Routes>
      <Route path="/" element={<RepoList />} />
      <Route path="/repos/:id" element={<RepoDetail />} />
    </Routes>
  );
}

export default App;