import { useState, useRef, useEffect } from "react";
import {
  Card,
  Input,
  Button,
  Space,
  Switch,
  Avatar,
  Typography,
  Tooltip,
  Tag,
  Empty,
  Spin,
  message,
  Collapse,
} from "antd";
import {
  SendOutlined,
  MessageOutlined,
  UserOutlined,
  RobotOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  ClearOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import { useRAGChat, conversationApi, ConversationMessage } from "@/lib/api";

const { TextArea } = Input;
const { Text, Paragraph } = Typography;
const { Panel } = Collapse;

export function ChatTab() {
  const { messages, isLoading, sendMessage, sendMessageStream, clearMessages } =
    useRAGChat();
  const [input, setInput] = useState("");
  const [filters, setFilters] = useState("");
  const [useStreaming, setUseStreaming] = useState(true);
  const [showThinking, setShowThinking] = useState(false);
  const [conversationId, setConversationId] = useState<string>("");
  const [isSaving, setIsSaving] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Initialize conversation ID on mount
  useEffect(() => {
    const initConversation = async () => {
      try {
        const { id } = await conversationApi.createNew();
        setConversationId(id);
      } catch (error) {
        console.error("Failed to create conversation:", error);
      }
    };
    initConversation();
  }, []);

  // Save conversation after each message exchange
  useEffect(() => {
    if (messages.length > 0 && conversationId && !isLoading) {
      const saveConversation = async () => {
        setIsSaving(true);
        try {
          const conversationMessages: ConversationMessage[] = messages.map(msg => ({
            role: msg.role,
            content: msg.content,
            sources: msg.sources,
            thinking: (msg as any).thinking,
            timestamp: Date.now()
          }));
          
          await conversationApi.save({
            id: conversationId,
            messages: conversationMessages,
          });
        } catch (error) {
          console.error("Failed to save conversation:", error);
        } finally {
          setIsSaving(false);
        }
      };
      saveConversation();
    }
  }, [messages, conversationId, isLoading]);

  const handleSubmit = async () => {
    if (!input.trim() || isLoading) return;

    const messageContent = input.trim();
    setInput("");

    // Parse filters if provided
    let parsedFilters: Record<string, any> | undefined;
    if (filters.trim()) {
      try {
        parsedFilters = JSON.parse(filters);
      } catch (error) {
        message.warning("Invalid filter JSON format");
        console.warn("Invalid filter JSON:", error);
      }
    }

    try {
      if (useStreaming) {
        await sendMessageStream(
          messageContent,
          parsedFilters,
          undefined,
          showThinking
        );
      } else {
        await sendMessage(messageContent, parsedFilters, showThinking);
      }
    } catch (error) {
      message.error("Failed to send message. Please try again.");
      console.error("Error sending message:", error);
    }
  };

  const handleClearMessages = async () => {
    clearMessages();
    // Create new conversation ID after clearing
    try {
      const { id } = await conversationApi.createNew();
      setConversationId(id);
      message.success("Conversation cleared and new session started");
    } catch (error) {
      message.error("Failed to create new conversation");
    }
  };

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      <Card
        title={
          <div style={{ textAlign: "left" }}>
            <Space>
              <MessageOutlined />
              <span>Chat with your Documents</span>
            </Space>
          </div>
        }
        extra={
          <Space>
            {conversationId && (
              <Tooltip title="Conversation ID">
                <Tag color="blue">ID: {conversationId.slice(0, 8)}</Tag>
              </Tooltip>
            )}
            {isSaving && (
              <Tag icon={<Spin size="small" />} color="processing">
                Saving...
              </Tag>
            )}
            <Tooltip title="Clear conversation">
              <Button
                icon={<ClearOutlined />}
                onClick={handleClearMessages}
                disabled={messages?.length === 0}
              >
                Clear
              </Button>
            </Tooltip>
          </Space>
        }
        style={{ flex: 1, display: "flex", flexDirection: "column" }}
      >
        <Text
          type="secondary"
          style={{ marginBottom: 16, textAlign: "left", display: "block" }}
        >
          Ask questions about your ingested documents and get AI-powered answers
          with sources.
        </Text>

        {/* Messages Area */}
        <div
          style={{
            flex: 1,
            overflowY: "auto",
            padding: 16,
            background: "#f5f5f5",
            borderRadius: 8,
            marginBottom: 16,
          }}
        >
          {messages?.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div
                  style={{
                    textAlign: "left",
                    width: "100%",
                    maxWidth: "600px",
                    margin: "0 auto",
                  }}
                >
                  Start a conversation by asking a question about your documents
                </div>
              }
            />
          ) : (
            <Space direction="vertical" style={{ width: "100%" }} size="middle">
              {messages?.map((msg, index) => (
                <div
                  key={index}
                  style={{
                    display: "flex",
                    justifyContent:
                      msg.role === "user" ? "flex-end" : "flex-start",
                  }}
                >
                  <Space
                    align="start"
                    style={{
                      maxWidth: "80%",
                      flexDirection:
                        msg.role === "user" ? "row-reverse" : "row",
                    }}
                  >
                    <Avatar
                      icon={
                        msg.role === "user" ? (
                          <UserOutlined />
                        ) : (
                          <RobotOutlined />
                        )
                      }
                      style={{
                        backgroundColor:
                          msg.role === "user" ? "#1890ff" : "#52c41a",
                      }}
                    />
                    <Card
                      size="small"
                      style={{
                        backgroundColor:
                          msg.role === "user" ? "#e6f7ff" : "#ffffff",
                        border:
                          msg.role === "user"
                            ? "1px solid #91d5ff"
                            : "1px solid #e8e8e8",
                      }}
                    >
                      <Paragraph
                        style={{
                          margin: 0,
                          whiteSpace: "pre-wrap",
                          textAlign: "left",
                        }}
                      >
                        {msg.content}
                      </Paragraph>

                      {/* Sources */}
                      {msg.sources && msg.sources.length > 0 && (
                        <div
                          style={{
                            marginTop: 12,
                            paddingTop: 12,
                            borderTop: "1px solid #f0f0f0",
                          }}
                        >
                          <Text
                            type="secondary"
                            style={{
                              fontSize: 12,
                              fontWeight: 500,
                              textAlign: "left",
                            }}
                          >
                            Sources:
                          </Text>
                          <Space
                            direction="vertical"
                            size="small"
                            style={{ width: "100%", marginTop: 8 }}
                          >
                            {msg.sources?.map((source: any, idx: number) => (
                              <Card
                                key={idx}
                                size="small"
                                style={{ backgroundColor: "#f9f9f9" }}
                                bodyStyle={{ padding: 8 }}
                              >
                                <Space>
                                  <FileTextOutlined />
                                  <Text style={{ fontSize: 12 }}>
                                    {source.source || source.id}
                                  </Text>
                                  <Tag color="blue" style={{ fontSize: 10 }}>
                                    Score: {(source.score * 100).toFixed(1)}%
                                  </Tag>
                                </Space>
                                {source.content && (
                                  <Paragraph
                                    style={{
                                      fontSize: 11,
                                      marginTop: 4,
                                      marginBottom: 0,
                                      color: "#666",
                                      textAlign: "left",
                                    }}
                                    ellipsis={{ rows: 2, expandable: true }}
                                  >
                                    {source.content}
                                  </Paragraph>
                                )}
                              </Card>
                            ))}
                          </Space>
                        </div>
                      )}

                      {/* Thinking */}
                      {(msg as any).thinking && (
                        <div
                          style={{
                            marginTop: 12,
                            paddingTop: 12,
                            borderTop: "1px solid #f0f0f0",
                          }}
                        >
                          <Collapse ghost size="small">
                            <Panel header="Show thinking process" key="1">
                              <Paragraph
                                style={{
                                  fontSize: 11,
                                  color: "#666",
                                  margin: 0,
                                  textAlign: "left",
                                }}
                              >
                                {(msg as any).thinking}
                              </Paragraph>
                            </Panel>
                          </Collapse>
                        </div>
                      )}
                    </Card>
                  </Space>
                </div>
              ))}
              {isLoading && (
                <div style={{ display: "flex", justifyContent: "flex-start" }}>
                  <Space align="start">
                    <Avatar
                      icon={<RobotOutlined />}
                      style={{ backgroundColor: "#52c41a" }}
                    />
                    <Card size="small">
                      <Spin size="small" />
                      <Text style={{ marginLeft: 8 }}>Thinking...</Text>
                    </Card>
                  </Space>
                </div>
              )}
              <div ref={messagesEndRef} />
            </Space>
          )}
        </div>

        {/* Input Area */}
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          {/* Advanced Options */}
          <Collapse ghost>
            <Panel
              header={
                <Space style={{ textAlign: "left" }}>
                  <SettingOutlined />
                  <Text style={{ textAlign: "left" }}>Advanced Options</Text>
                </Space>
              }
              key="1"
            >
              <Space direction="vertical" style={{ width: "100%" }}>
                <Space>
                  <Text style={{ textAlign: "left" }}>Streaming Mode:</Text>
                  <Switch
                    checked={useStreaming}
                    onChange={setUseStreaming}
                    checkedChildren={<ThunderboltOutlined />}
                    unCheckedChildren="Off"
                  />
                </Space>
                <Space>
                  <Text style={{ textAlign: "left" }}>Show Thinking:</Text>
                  <Switch
                    checked={showThinking}
                    onChange={setShowThinking}
                    checkedChildren="On"
                    unCheckedChildren="Off"
                  />
                </Space>
                <div>
                  <Text style={{ textAlign: "left" }}>Filters (JSON):</Text>
                  <Input
                    placeholder='{"category": "medical", "date": "2024"}'
                    value={filters}
                    onChange={(e) => setFilters(e.target.value)}
                    style={{ marginTop: 8 }}
                  />
                </div>
              </Space>
            </Panel>
          </Collapse>

          {/* Message Input */}
          <Space.Compact style={{ width: "100%" }}>
            <TextArea
              placeholder="Ask a question about your documents..."
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onPressEnter={(e) => {
                if (!e.shiftKey) {
                  e.preventDefault();
                  handleSubmit();
                }
              }}
              autoSize={{ minRows: 2, maxRows: 6 }}
              style={{ flex: 1 }}
              disabled={isLoading}
            />
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={handleSubmit}
              loading={isLoading}
              style={{ height: "auto" }}
            >
              Send
            </Button>
          </Space.Compact>
        </Space>
      </Card>
    </div>
  );
}
