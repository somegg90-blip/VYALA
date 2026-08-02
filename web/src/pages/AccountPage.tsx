import DashboardLayout from '../components/DashboardLayout';

export default function AccountPage() {
  return (
    <DashboardLayout>
      <h1 className="text-3xl font-bold mb-8">Account Settings</h1>
      <div className="bg-white p-6 rounded shadow border max-w-lg">
        <h2 className="text-xl font-semibold mb-4">Subscription</h2>
        <p className="text-gray-600">You are currently on the <span className="font-bold text-blue-600">Free Tier</span>.</p>
        <button disabled className="mt-4 bg-gray-200 text-gray-500 px-4 py-2 rounded cursor-not-allowed">
          Upgrade to Team (Coming Soon)
        </button>
      </div>
    </DashboardLayout>
  );
}