package com.workflow;

import java.util.*;
import java.util.concurrent.*;
import java.util.function.Function;
import java.util.function.Predicate;

/**
 * 可配置的工作流引擎
 * 支持任务节点、条件分支、并行执行
 * 支持状态持久化和恢复
 */
public class WorkflowEngine {

    private final Map<String, Workflow> workflows = new ConcurrentHashMap<>();
    private final Map<String, WorkflowState> workflowStates = new ConcurrentHashMap<>();
    private final ExecutorService executorService = Executors.newFixedThreadPool(10);

    /**
     * 执行工作流
     * @param workflow 要执行的工作流
     * @return 工作流 ID
     */
    public String execute(Workflow workflow) {
        if (workflow == null || workflow.getId() == null) {
            throw new IllegalArgumentException("Workflow and its ID must not be null");
        }

        String workflowId = workflow.getId();
        workflows.put(workflowId, workflow);

        WorkflowState state = new WorkflowState(workflowId);
        workflowStates.put(workflowId, state);

        // 异步执行工作流
        CompletableFuture.run(() -> {
            try {
                executeWorkflow(workflow, state);
            } catch (Exception e) {
                state.setStatus(WorkflowStatus.FAILED);
                state.setError(e.getMessage());
            }
        });

        return workflowId;
    }

    /**
     * 暂停工作流
     * @param workflowId 工作流 ID
     * @return 是否成功暂停
     */
    public boolean pause(String workflowId) {
        WorkflowState state = workflowStates.get(workflowId);
        if (state == null) {
            return false;
        }

        synchronized (state) {
            if (state.getStatus() == WorkflowStatus.RUNNING) {
                state.setStatus(WorkflowStatus.PAUSED);
                state.setPausedAt(System.currentTimeMillis());
                return true;
            }
        }
        return false;
    }

    /**
     * 恢复工作流
     * @param workflowId 工作流 ID
     * @return 是否成功恢复
     */
    public boolean resume(String workflowId) {
        WorkflowState state = workflowStates.get(workflowId);
        if (state == null) {
            return false;
        }

        synchronized (state) {
            if (state.getStatus() == WorkflowStatus.PAUSED) {
                state.setStatus(WorkflowStatus.RUNNING);
                state.setPausedAt(null);
                
                // 恢复执行
                Workflow workflow = workflows.get(workflowId);
                CompletableFuture.run(() -> executeWorkflow(workflow, state));
                return true;
            }
        }
        return false;
    }

    /**
     * 获取工作流状态
     * @param workflowId 工作流 ID
     * @return 工作流状态信息
     */
    public WorkflowState getStatus(String workflowId) {
        return workflowStates.get(workflowId);
    }

    /**
     * 执行工作流的内部方法
     */
    private void executeWorkflow(Workflow workflow, WorkflowState state) {
        List<Node> nodes = workflow.getNodes();
        if (nodes == null || nodes.isEmpty()) {
            state.setStatus(WorkflowStatus.COMPLETED);
            return;
        }

        int currentIndex = 0;
        
        while (currentIndex < nodes.size()) {
            // 检查是否暂停
            if (state.getStatus() == WorkflowStatus.PAUSED) {
                try {
                    waitUntilResumed(state);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    break;
                }
            }

            Node currentNode = nodes.get(currentIndex);
            
            try {
                executeNode(currentNode, state);
                
                // 处理并行节点
                if (currentNode.getType() == NodeType.PARALLEL) {
                    currentIndex = handleParallelNode((ParallelNode) currentNode, state, nodes, currentIndex);
                } 
                // 处理条件分支
                else if (currentNode.getType() == NodeType.CONDITION) {
                    currentIndex = handleConditionNode((ConditionNode) currentNode, state, nodes, currentIndex);
                }
                
                currentIndex++;
                
            } catch (Exception e) {
                state.setStatus(WorkflowStatus.FAILED);
                state.setError(e.getMessage());
                break;
            }
        }

        if (state.getStatus() != WorkflowStatus.FAILED) {
            state.setStatus(WorkflowStatus.COMPLETED);
        }
    }

    /**
     * 执行单个节点
     */
    private void executeNode(Node node, WorkflowState state) {
        state.setCurrentNodeId(node.getId());
        state.setStatus(WorkflowStatus.RUNNING);

        switch (node.getType()) {
            case TASK:
                TaskNode taskNode = (TaskNode) node;
                taskNode.getTaskFunction().apply(state.getContext());
                break;
                
            case CONDITION:
                // 条件节点的处理在外部进行
                break;
                
            case PARALLEL:
                // 并行节点的处理在外部进行
                break;
        }

        state.setCompletedNodes(state.getCompletedNodes() + 1);
    }

    /**
     * 处理并行节点
     */
    private int handleParallelNode(ParallelNode parallelNode, WorkflowState state, 
                                   List<Node> nodes, int currentIndex) throws Exception {
        
        List<Node> parallelBranches = parallelNode.getBranches();
        List<CompletableFuture<Void>> futures = new ArrayList<>();

        for (Node branch : parallelBranches) {
            CompletableFuture<Void> future = CompletableFuture.runAsync(() -> {
                try {
                    executeNode(branch, state);
                } catch (Exception e) {
                    state.setError(e.getMessage());
                }
            }, executorService);
            futures.add(future);
        }

        // 等待所有并行分支完成
        CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).join();

        // 返回下一个节点索引（跳过并行节点及其分支）
        return currentIndex + parallelNode.getBranches().size();
    }

    /**
     * 处理条件分支节点
     */
    private int handleConditionNode(ConditionNode conditionNode, WorkflowState state, 
                                    List<Node> nodes, int currentIndex) {
        
        // 根据条件表达式选择下一个分支
        String condition = conditionNode.getCondition();
        boolean conditionMet = evaluateCondition(condition, state.getContext());

        if (conditionMet && conditionNode.getTrueBranch() != null) {
            executeNode(conditionNode.getTrueBranch(), state);
            return currentIndex + 1;
        } else if (!conditionMet && conditionNode.getFalseBranch() != null) {
            executeNode(conditionNode.getFalseBranch(), state);
            return currentIndex + 1;
        }

        return currentIndex + 1;
    }

    /**
     * 评估条件表达式
     */
    private boolean evaluateCondition(String condition, Map<String, Object> context) {
        // 简单的条件评估实现
        // 支持格式：context.key == value 或 context.key != null
        if (condition.contains("==")) {
            String[] parts = condition.split("==");
            String key = parts[0].trim().replace("context.", "");
            Object expectedValue = parseValue(parts[1].trim());
            return Objects.equals(context.get(key), expectedValue);
        } else if (condition.contains("!= null")) {
            String key = condition.replace("context.", "").replace("!= null", "").trim();
            return context.containsKey(key) && context.get(key) != null;
        }
        return true;
    }

    /**
     * 等待工作流恢复
     */
    private void waitUntilResumed(WorkflowState state) throws InterruptedException {
        while (state.getStatus() == WorkflowStatus.PAUSED) {
            synchronized (state) {
                state.wait(1000);
            }
        }
    }

    /**
     * 解析值（支持字符串、数字、布尔值）
     */
    private Object parseValue(String value) {
        value = value.trim();
        
        // 去除引号
        if (value.startsWith("\"") && value.endsWith("\"")) {
            return value.substring(1, value.length() - 1);
        }
        
        // 尝试解析为数字
        try {
            if (value.contains(".")) {
                return Double.parseDouble(value);
            }
            return Long.parseLong(value);
        } catch (NumberFormatException e) {
            // 尝试解析布尔值
            if (value.equalsIgnoreCase("true")) {
                return true;
            } else if (value.equalsIgnoreCase("false")) {
                return false;
            }
            return value;
        }
    }

    /**
     * 持久化工作流状态
     */
    public void persistState(String workflowId) {
        WorkflowState state = workflowStates.get(workflowId);
        if (state != null) {
            // 这里可以扩展为写入文件或数据库
            state.setLastSavedAt(System.currentTimeMillis());
        }
    }

    /**
     * 从持久化状态恢复工作流
     */
    public void restoreState(String workflowId, WorkflowState savedState) {
        if (savedState != null) {
            workflowStates.put(workflowId, savedState);
        }
    }

    /**
     * 关闭工作流引擎
     */
    public void shutdown() {
        executorService.shutdown();
        try {
            if (!executorService.awaitTermination(60, TimeUnit.SECONDS)) {
                executorService.shutdownNow();
            }
        } catch (InterruptedException e) {
            executorService.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    // ==================== 内部类定义 ====================

    /**
     * 工作流类
     */
    public static class Workflow {
        private String id;
        private String name;
        private List<Node> nodes;

        public Workflow() {}

        public Workflow(String id, String name) {
            this.id = id;
            this.name = name;
            this.nodes = new ArrayList<>();
        }

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }
        public String getName() { return name; }
        public void setName(String name) { this.name = name; }
        public List<Node> getNodes() { return nodes; }
        public void setNodes(List<Node> nodes) { this.nodes = nodes; }
        public void addNode(Node node) { this.nodes.add(node); }
    }

    /**
     * 节点类型枚举
     */
    public enum NodeType {
        TASK,      // 任务节点
        CONDITION, // 条件分支
        PARALLEL   // 并行执行
    }

    /**
     * 基础节点类
     */
    public static class Node {
        private String id;
        private NodeType type;

        public Node() {}

        public Node(String id, NodeType type) {
            this.id = id;
            this.type = type;
        }

        public String getId() { return id; }
        public void setId(String id) { this.id = id; }
        public NodeType getType() { return type; }
        public void setType(NodeType type) { this.type = type; }
    }

    /**
     * 任务节点类
     */
    public static class TaskNode extends Node {
        private Function<Map<String, Object>, Void> taskFunction;

        public TaskNode() {}

        public TaskNode(String id, Function<Map<String, Object>, Void> taskFunction) {
            super(id, NodeType.TASK);
            this.taskFunction = taskFunction;
        }

        public Function<Map<String, Object>, Void> getTaskFunction() { return taskFunction; }
        public void setTaskFunction(Function<Map<String, Object>, Void> taskFunction) { 
            this.taskFunction = taskFunction; 
        }
    }

    /**
     * 条件节点类
     */
    public static class ConditionNode extends Node {
        private String condition;
        private Node trueBranch;
        private Node falseBranch;

        public ConditionNode() {}

        public ConditionNode(String id, String condition) {
            super(id, NodeType.CONDITION);
            this.condition = condition;
        }

        public String getCondition() { return condition; }
        public void setCondition(String condition) { this.condition = condition; }
        public Node getTrueBranch() { return trueBranch; }
        public void setTrueBranch(Node trueBranch) { this.trueBranch = trueBranch; }
        public Node getFalseBranch() { return falseBranch; }
        public void setFalseBranch(Node falseBranch) { this.falseBranch = falseBranch; }
    }

    /**
     * 并行节点类
     */
    public static class ParallelNode extends Node {
        private List<Node> branches;

        public ParallelNode() {}

        public ParallelNode(String id) {
            super(id, NodeType.PARALLEL);
            this.branches = new ArrayList<>();
        }

        public List<Node> getBranches() { return branches; }
        public void setBranches(List<Node> branches) { this.branches = branches; }
        public void addBranch(Node branch) { this.branches.add(branch); }
    }

    /**
     * 工作流状态枚举
     */
    public enum WorkflowStatus {
        PENDING,    // 待执行
        RUNNING,    // 运行中
        PAUSED,     // 已暂停
        COMPLETED,  // 已完成
        FAILED      // 失败
    }

    /**
     * 工作流状态类
     */
    public static class WorkflowState {
        private String workflowId;
        private WorkflowStatus status;
        private String currentNodeId;
        private int completedNodes;
        private long startTime;
        private Long pausedAt;
        private Long lastSavedAt;
        private String error;
        private Map<String, Object> context = new HashMap<>();

        public WorkflowState() {}

        public WorkflowState(String workflowId) {
            this.workflowId = workflowId;
            this.status = WorkflowStatus.PENDING;
            this.startTime = System.currentTimeMillis();
        }

        public String getWorkflowId() { return workflowId; }
        public void setWorkflowId(String workflowId) { this.workflowId = workflowId; }
        public WorkflowStatus getStatus() { return status; }
        public void setStatus(WorkflowStatus status) { this.status = status; }
        public String getCurrentNodeId() { return currentNodeId; }
        public void setCurrentNodeId(String currentNodeId) { this.currentNodeId = currentNodeId; }
        public int getCompletedNodes() { return completedNodes; }
        public void setCompletedNodes(int completedNodes) { this.completedNodes = completedNodes; }
        public long getStartTime() { return startTime; }
        public void setStartTime(long startTime) { this.startTime = startTime; }
        public Long getPausedAt() { return pausedAt; }
        public void setPausedAt(Long pausedAt) { this.pausedAt = pausedAt; }
        public Long getLastSavedAt() { return lastSavedAt; }
        public void setLastSavedAt(Long lastSavedAt) { this.lastSavedAt = lastSavedAt; }
        public String getError() { return error; }
        public void setError(String error) { this.error = error; }
        public Map<String, Object> getContext() { return context; }
        public void setContext(Map<String, Object> context) { this.context = context; }
    }

    // ==================== 示例用法 ====================
    
    public static void main(String[] args) {
        WorkflowEngine engine = new WorkflowEngine();

        // 创建工作流
        Workflow workflow = new Workflow("wf-001", "Sample Workflow");

        // 添加任务节点
        TaskNode task1 = new TaskNode("task-1", (context) -> {
            System.out.println("Executing task-1");
            context.put("result1", "Task 1 completed");
            return null;
        });

        // 添加条件节点
        ConditionNode condition = new ConditionNode("cond-1", "context.result1 != null");
        TaskNode trueTask = new TaskNode("task-true", (ctx) -> {
            System.out.println("Executing true branch task");
            return null;
        });
        TaskNode falseTask = new TaskNode("task-false", (ctx) -> {
            System.out.println("Executing false branch task");
            return null;
        });
        condition.setTrueBranch(trueTask);
        condition.setFalseBranch(falseTask);

        // 添加并行节点
        ParallelNode parallel = new ParallelNode("parallel-1");
        TaskNode parallelTask1 = new TaskNode("p-task-1", (ctx) -> {
            System.out.println("Executing parallel task 1");
            return null;
        });
        TaskNode parallelTask2 = new TaskNode("p-task-2", (ctx) -> {
            System.out.println("Executing parallel task 2");
            return null;
        });
        parallel.addBranch(parallelTask1);
        parallel.addBranch(parallelTask2);

        // 组装工作流
        workflow.addNode(task1);
        workflow.addNode(condition);
        workflow.addNode(parallel);

        // 执行工作流
        String workflowId = engine.execute(workflow);
        System.out.println("Workflow started with ID: " + workflowId);

        // 获取状态
        WorkflowState state = engine.getStatus(workflowId);
        System.out.println("Initial status: " + state.getStatus());

        // 暂停工作流
        boolean paused = engine.pause(workflowId);
        System.out.println("Paused: " + paused);

        // 恢复工作流
        boolean resumed = engine.resume(workflowId);
        System.out.println("Resumed: " + resumed);

        // 等待执行完成
        try {
            Thread.sleep(3000);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }

        // 获取最终状态
        WorkflowState finalState = engine.getStatus(workflowId);
        System.out.println("Final status: " + finalState.getStatus());
        System.out.println("Completed nodes: " + finalState.getCompletedNodes());

        // 持久化状态
        engine.persistState(workflowId);

        // 关闭引擎
        engine.shutdown();
    }
}
