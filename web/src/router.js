import { createRouter, createWebHistory } from "vue-router";
import AutomationPage from "./pages/AutomationPage.vue";
import DashboardPage from "./pages/DashboardPage.vue";
import JobDetailPage from "./pages/JobDetailPage.vue";
import JobPage from "./pages/JobPage.vue";
import ProviderPage from "./pages/ProviderPage.vue";

const routes = [
    {
        path: "/",
        redirect: "/dashboard",
    },
    {
        path: "/dashboard",
        name: "dashboard",
        component: DashboardPage,
    },
    {
        path: "/providers",
        name: "providers",
        component: ProviderPage,
    },
    {
        path: "/automations",
        name: "automations",
        component: AutomationPage,
    },
    {
        path: "/jobs",
        name: "jobs",
        component: JobPage,
    },
    {
        path: "/automations/:automationId/jobs/:jobId",
        name: "job-detail",
        component: JobDetailPage,
    },
];

export const router = createRouter({
    history: createWebHistory(),
    routes,
});