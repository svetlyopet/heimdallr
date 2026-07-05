import { createRouter, createWebHistory } from "vue-router";
import ApplicationDetailPage from "./pages/ApplicationDetailPage.vue";
import ApplicationPage from "./pages/ApplicationPage.vue";
import AutomationPage from "./pages/AutomationPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import JobDetailPage from "./pages/JobDetailPage.vue";
import JobPage from "./pages/JobPage.vue";
import LoginPage from "./pages/LoginPage.vue";
import ProviderPage from "./pages/ProviderPage.vue";
import ReleaseDetailPage from "./pages/ReleaseDetailPage.vue";
import ReportDetailPage from "./pages/ReportDetailPage.vue";
import UsersPage from "./pages/UsersPage.vue";
import { ensureSessionAccess, sessionState } from "./auth/session";

const routes = [
    {
        path: "/",
        redirect: "/login",
    },
    {
        path: "/login",
        name: "login",
        component: LoginPage,
        meta: {
            guestOnly: true,
        },
    },
    {
        path: "/dashboard",
        name: "dashboard",
        component: DashboardPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/applications",
        name: "applications",
        component: ApplicationPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/applications/:id",
        name: "application-detail",
        component: ApplicationDetailPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/applications/:id/releases/:releaseId",
        name: "release-detail",
        component: ReleaseDetailPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/applications/:id/releases/:releaseId/reports/:reportId",
        name: "report-detail",
        component: ReportDetailPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/providers",
        name: "providers",
        component: ProviderPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/automations",
        name: "automations",
        component: AutomationPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/jobs",
        name: "jobs",
        component: JobPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/automations/:automationId/jobs/:jobId",
        name: "job-detail",
        component: JobDetailPage,
        meta: {
            requiresAuth: true,
        },
    },
    {
        path: "/users",
        name: "users",
        component: UsersPage,
        meta: {
            requiresAuth: true,
            requiresAdmin: true,
        },
    },
];

export const router = createRouter({
    history: createWebHistory(),
    routes,
});

router.beforeEach(async (to) => {
    await ensureSessionAccess();

    if (to.meta.requiresAuth && !sessionState.authenticated) {
        return { name: "login" };
    }

    if (to.meta.guestOnly && sessionState.authenticated) {
        return { name: "dashboard" };
    }

    if (to.meta.requiresAdmin && !sessionState.roles.includes("admin")) {
        return { name: "dashboard" };
    }

    return true;
});
