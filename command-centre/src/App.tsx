/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */
import React, { Suspense, lazy } from 'react';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { SessionProvider } from './lib/session';
import { AppShell } from './components/layout/AppShell';
import { Loading } from './components/common/states';

const DashboardView = lazy(() => import('./components/pages/DashboardView'));
const AppsView = lazy(() => import('./components/pages/AppsView'));
const AppDetailView = lazy(() => import('./components/pages/AppDetailView'));
const DeployView = lazy(() => import('./components/pages/DeployView'));
const DeployNewView = lazy(() => import('./components/pages/DeployNewView'));
const DeploymentView = lazy(() => import('./components/pages/DeploymentView'));
const NodesView = lazy(() => import('./components/pages/NodesView'));
const NodeDetailView = lazy(() => import('./components/pages/NodeDetailView'));
const AddNodeView = lazy(() => import('./components/pages/AddNodeView'));
const StorageView = lazy(() => import('./components/pages/StorageView'));
const DomainsView = lazy(() => import('./components/pages/DomainsView'));
const SecurityView = lazy(() => import('./components/pages/SecurityView'));
const AnalyticsView = lazy(() => import('./components/pages/AnalyticsView'));
const EvidenceView = lazy(() => import('./components/pages/EvidenceView'));
const EvidenceDetailView = lazy(() => import('./components/pages/EvidenceDetailView'));
const BillingView = lazy(() => import('./components/pages/BillingView'));
const TeamView = lazy(() => import('./components/pages/TeamView'));
const SettingsView = lazy(() => import('./components/pages/SettingsView'));
const ActivityView = lazy(() => import('./components/pages/ActivityView'));
const OperationsView = lazy(() => import('./components/pages/OperationsView'));
const CopilotView = lazy(() => import('./components/pages/CopilotView'));
const NotFoundView = lazy(() => import('./components/pages/NotFoundView'));

export default function App() {
  return (
    <BrowserRouter>
      <SessionProvider>
        <AppShell>
          <Suspense fallback={<Loading />}>
            <Routes>
              <Route path="/" element={<DashboardView />} />
              <Route path="/dashboard" element={<Navigate to="/" replace />} />
              <Route path="/apps" element={<AppsView />} />
              <Route path="/apps/:name" element={<AppDetailView />} />
              <Route path="/deploy" element={<DeployView />} />
              <Route path="/deploy/new" element={<DeployNewView />} />
              <Route path="/deploy/:app/:generation" element={<DeploymentView />} />
              <Route path="/nodes" element={<NodesView />} />
              <Route path="/nodes/add" element={<AddNodeView />} />
              <Route path="/nodes/:id" element={<NodeDetailView />} />
              <Route path="/storage" element={<StorageView />} />
              <Route path="/domains" element={<DomainsView />} />
              <Route path="/security" element={<SecurityView />} />
              <Route path="/analytics" element={<AnalyticsView />} />
              <Route path="/evidence" element={<EvidenceView />} />
              <Route path="/evidence/:id" element={<EvidenceDetailView />} />
              <Route path="/billing" element={<BillingView />} />
              <Route path="/team" element={<TeamView />} />
              <Route path="/copilot" element={<CopilotView />} />
              <Route path="/settings" element={<SettingsView />} />
              <Route path="/activity" element={<ActivityView />} />
              <Route path="/operations" element={<OperationsView />} />
              <Route path="*" element={<NotFoundView />} />
            </Routes>
          </Suspense>
        </AppShell>
      </SessionProvider>
    </BrowserRouter>
  );
}
