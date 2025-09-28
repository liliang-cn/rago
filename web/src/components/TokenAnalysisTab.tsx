import { useState, useEffect } from "react";
import {
  Card,
  Button,
  Tag,
  Tabs,
  Select,
  Alert,
  Space,
  Typography,
  Statistic,
  Row,
  Col,
  Empty,
} from "antd";
import {
  BarChartOutlined,
  DollarOutlined,
  DashboardOutlined,
  GoldOutlined,
  ExclamationCircleOutlined,
  CalendarOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  apiClient,
  UsageStats,
  UsageStatsByType,
  UsageStatsByProvider,
} from "@/lib/api";

const { Title, Text } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

export function TokenAnalysisTab() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [overallStats, setOverallStats] = useState<UsageStats | null>(null);
  const [statsByType, setStatsByType] = useState<UsageStatsByType | null>(null);
  const [statsByProvider, setStatsByProvider] =
    useState<UsageStatsByProvider | null>(null);
  const [dailyUsage, setDailyUsage] = useState<Record<
    string,
    UsageStats
  > | null>(null);
  const [topModels, setTopModels] = useState<Record<string, number> | null>(
    null
  );
  const [costs, setCosts] = useState<Record<string, number> | null>(null);
  const [timeRange, setTimeRange] = useState<string>("all");

  const timeRanges = {
    all: { label: "All Time", days: 0 },
    "1d": { label: "1 Day", days: 1 },
    "7d": { label: "7 Days", days: 7 },
    "30d": { label: "30 Days", days: 30 },
    "90d": { label: "90 Days", days: 90 },
  };

  console.log(overallStats, "overallStats");

  const fetchData = async () => {
    setLoading(true);
    setError(null);

    try {
      const now = new Date();
      const startTime = new Date(
        now.getTime() -
          timeRanges[timeRange as keyof typeof timeRanges].days *
            24 *
            60 *
            60 *
            1000
      );

      // Create filter for time range  
      const filter =
        timeRange === "all"
          ? {}
          : {
              start_time: Math.floor(startTime.getTime() / 1000).toString(),
              end_time: Math.floor(now.getTime() / 1000).toString(),
            };

      // Fetch all data in parallel
      const [
        overallResponse,
        typeResponse,
        providerResponse,
        dailyResponse,
        modelsResponse,
        costsResponse,
      ] = await Promise.all([
        apiClient.getUsageStats(filter),
        apiClient.getUsageStatsByType(filter),
        apiClient.getUsageStatsByProvider(filter),
        apiClient.getDailyUsage(
          timeRanges[timeRange as keyof typeof timeRanges].days
        ),
        apiClient.getTopModels(),
        apiClient.getCostByProvider(),
      ]);

      // Handle errors gracefully
      if (overallResponse.error) {
        console.warn("Overall stats error:", overallResponse.error);
      } else {
        setOverallStats(
          (overallResponse.data as any)?.data || overallResponse.data || null
        );
      }

      if (typeResponse.error) {
        console.warn("Type stats error:", typeResponse.error);
      } else {
        setStatsByType(
          (typeResponse.data as any)?.data || typeResponse.data || null
        );
      }

      if (providerResponse.error) {
        console.warn("Provider stats error:", providerResponse.error);
      } else {
        setStatsByProvider(
          (providerResponse.data as any)?.data || providerResponse.data || null
        );
      }

      if (dailyResponse.error) {
        console.warn("Daily usage error:", dailyResponse.error);
      } else {
        setDailyUsage(
          (dailyResponse.data as any)?.data || dailyResponse.data || null
        );
      }

      if (modelsResponse.error) {
        console.warn("Top models error:", modelsResponse.error);
      } else {
        setTopModels(
          (modelsResponse.data as any)?.data || modelsResponse.data || null
        );
      }

      if (costsResponse.error) {
        console.warn("Costs error:", costsResponse.error);
      } else {
        setCosts(
          (costsResponse.data as any)?.data || costsResponse.data || null
        );
      }
    } catch (err) {
      console.error("Fetch data error:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch usage data"
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [timeRange]);

  const formatNumber = (num: number) => {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(1)}M`;
    }
    if (num >= 1000) {
      return `${(num / 1000).toFixed(1)}K`;
    }
    return num.toString();
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 4,
    }).format(amount);
  };

  if (error) {
    return (
      <Alert
        message={error}
        type="error"
        showIcon
        icon={<ExclamationCircleOutlined />}
        style={{ margin: 16 }}
      />
    );
  }

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      <Space direction="vertical" style={{ width: "100%" }} size="large">
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <div>
            <Title level={2} style={{ margin: 0 }}>
              Token Analysis
            </Title>
            <Text type="secondary">
              Monitor token consumption, costs, and usage patterns
            </Text>
          </div>
          <Space>
            <Select
              value={timeRange}
              onChange={setTimeRange}
              style={{ width: 120 }}
            >
              {Object.entries(timeRanges).map(([key, range]) => (
                <Option key={key} value={key}>
                  {range.label}
                </Option>
              ))}
            </Select>
            <Button
              onClick={fetchData}
              disabled={loading}
              loading={loading}
              icon={<ReloadOutlined />}
            >
              Refresh
            </Button>
          </Space>
        </div>

        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="Total Tokens"
                value={overallStats?.total_tokens || 0}
                prefix={<BarChartOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="Total Cost"
                value={overallStats?.total_cost || 0}
                prefix={<DollarOutlined />}
                formatter={(value) => `$${value}`}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="Requests"
                value={overallStats?.total_calls || 0}
                prefix={<DashboardOutlined />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Card>
              <Statistic
                title="Avg Response Time"
                value={overallStats?.average_latency || 0}
                suffix="ms"
                prefix={<GoldOutlined />}
              />
            </Card>
          </Col>
        </Row>

        <Tabs defaultActiveKey="byType" type="card">
          <TabPane
            tab={
              <Space>
                <BarChartOutlined />
                By Type
              </Space>
            }
            key="byType"
          >
            <Card title="Usage by Type" loading={loading}>
              {statsByType ? (
                <Row gutter={[16, 16]}>
                  {Object.entries(statsByType).map(([type, stats]) => (
                    <Col xs={24} sm={12} md={8} key={type}>
                      <Card size="small">
                        <Statistic
                          title={type.charAt(0).toUpperCase() + type.slice(1)}
                          value={stats.total_tokens || 0}
                          suffix="tokens"
                          prefix={<BarChartOutlined />}
                        />
                        <div
                          style={{ marginTop: 8, fontSize: 12, color: "#666" }}
                        >
                          <div>Cost: ${stats.total_cost || 0}</div>
                          <div>Requests: {stats.total_calls || 0}</div>
                        </div>
                      </Card>
                    </Col>
                  ))}
                </Row>
              ) : (
                <Empty description="No usage data by type available" />
              )}
            </Card>
          </TabPane>

          <TabPane
            tab={
              <Space>
                <DashboardOutlined />
                By Provider
              </Space>
            }
            key="byProvider"
          >
            <Card title="Usage by Provider" loading={loading}>
              {statsByProvider ? (
                <Row gutter={[16, 16]}>
                  {Object.entries(statsByProvider).map(([provider, stats]) => (
                    <Col xs={24} sm={12} md={8} key={provider}>
                      <Card size="small">
                        <Statistic
                          title={provider}
                          value={stats.total_tokens || 0}
                          suffix="tokens"
                          prefix={<DashboardOutlined />}
                          formatter={(value) =>
                            formatNumber(Number(value) || 0)
                          }
                        />
                        <div
                          style={{ marginTop: 8, fontSize: 12, color: "#666" }}
                        >
                          <div>
                            Cost: {formatCurrency(stats.total_cost || 0)}
                          </div>
                          <div>Requests: {stats.total_calls || 0}</div>
                        </div>
                      </Card>
                    </Col>
                  ))}
                </Row>
              ) : (
                <Empty description="No usage data by provider available" />
              )}
            </Card>
          </TabPane>

          <TabPane
            tab={
              <Space>
                <CalendarOutlined />
                Daily Usage
              </Space>
            }
            key="daily"
          >
            <Card title="Daily Usage Trends" loading={loading}>
              {dailyUsage ? (
                <Space
                  direction="vertical"
                  style={{ width: "100%" }}
                  size="middle"
                >
                  {Object.entries(dailyUsage).map(([date, stats]) => (
                    <div
                      key={date}
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                        padding: 12,
                        backgroundColor: "#fafafa",
                        borderRadius: 6,
                        border: "1px solid #f0f0f0",
                      }}
                    >
                      <Space>
                        <CalendarOutlined />
                        <Text strong>
                          {new Date(date).toLocaleDateString()}
                        </Text>
                      </Space>
                      <Space size="large">
                        <div style={{ textAlign: "center" }}>
                          <div
                            style={{
                              fontSize: 16,
                              fontWeight: "bold",
                              color: "#1890ff",
                            }}
                          >
                            {formatNumber(stats.total_tokens || 0)}
                          </div>
                          <div style={{ fontSize: 12, color: "#666" }}>
                            tokens
                          </div>
                        </div>
                        <div style={{ textAlign: "center" }}>
                          <div
                            style={{
                              fontSize: 16,
                              fontWeight: "bold",
                              color: "#52c41a",
                            }}
                          >
                            {formatCurrency(stats.total_cost || 0)}
                          </div>
                          <div style={{ fontSize: 12, color: "#666" }}>
                            cost
                          </div>
                        </div>
                        <div style={{ textAlign: "center" }}>
                          <div
                            style={{
                              fontSize: 16,
                              fontWeight: "bold",
                              color: "#722ed1",
                            }}
                          >
                            {stats.total_calls || 0}
                          </div>
                          <div style={{ fontSize: 12, color: "#666" }}>
                            requests
                          </div>
                        </div>
                      </Space>
                    </div>
                  ))}
                </Space>
              ) : (
                <Empty description="No daily usage data available" />
              )}
            </Card>
          </TabPane>

          <TabPane
            tab={
              <Space>
                <GoldOutlined />
                Top Models
              </Space>
            }
            key="models"
          >
            <Card title="Most Used Models" loading={loading}>
              {topModels ? (
                <Space
                  direction="vertical"
                  style={{ width: "100%" }}
                  size="small"
                >
                  {Object.entries(topModels).map(([model, count]) => (
                    <div
                      key={model}
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                        padding: 8,
                        backgroundColor: "#fafafa",
                        borderRadius: 4,
                        border: "1px solid #f0f0f0",
                      }}
                    >
                      <Space>
                        <GoldOutlined style={{ color: "#faad14" }} />
                        <Text>{model}</Text>
                      </Space>
                      <Tag color="blue">{count} uses</Tag>
                    </div>
                  ))}
                </Space>
              ) : (
                <Empty description="No model usage data available" />
              )}
            </Card>
          </TabPane>

          <TabPane
            tab={
              <Space>
                <DollarOutlined />
                Costs
              </Space>
            }
            key="costs"
          >
            <Card title="Cost Breakdown" loading={loading}>
              {costs ? (
                <Row gutter={[16, 16]}>
                  {Object.entries(costs).map(([provider, cost]) => (
                    <Col xs={24} sm={12} md={8} key={provider}>
                      <Card size="small">
                        <Statistic
                          title={provider}
                          value={cost || 0}
                          prefix={<DollarOutlined />}
                          formatter={(value) =>
                            formatCurrency(Number(value) || 0)
                          }
                          valueStyle={{ color: "#f5222d" }}
                        />
                      </Card>
                    </Col>
                  ))}
                </Row>
              ) : (
                <Empty description="No cost data available" />
              )}
            </Card>
          </TabPane>
        </Tabs>
      </Space>
    </div>
  );
}
