// ChatGPT Clone - JavaScript

const API_URL = '/api/v1/chat';

// State
let messages = [];
let selectedModel = 'llama3.2:1b';
let isLoading = false;

// Model display names
const MODEL_NAMES = {
    'llama3.2:1b': 'Llama 3.2 (1B)',
    'llama3.2:3b': 'Llama 3.2 (3B)',
    'mistral:7b': 'Mistral (7B)',
    'phi3:3.8b': 'Phi-3 (3.8B)',
    'gemma2:2b': 'Gemma 2 (2B)',
    'qwen2.5:3b': 'Qwen 2.5 (3B)'
};

// DOM Elements
const chatContainer = document.getElementById('chat-container');
const messagesContainer = document.getElementById('messages');
const welcomeMessage = document.getElementById('welcome-message');
const messageInput = document.getElementById('message-input');
const sendBtn = document.getElementById('send-btn');
const modelSelect = document.getElementById('model-select');
const currentModelName = document.getElementById('current-model-name');
const messageCount = document.getElementById('message-count');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    // Load saved state from localStorage
    loadState();
    updateUI();

    // Enable send button when there's input
    messageInput.addEventListener('input', () => {
        sendBtn.disabled = messageInput.value.trim() === '' || isLoading;
    });
});

// Load state from localStorage
function loadState() {
    try {
        const savedMessages = localStorage.getItem('chatMessages');
        const savedModel = localStorage.getItem('selectedModel');

        if (savedMessages) {
            messages = JSON.parse(savedMessages);
        }
        if (savedModel) {
            selectedModel = savedModel;
            modelSelect.value = selectedModel;
        }
    } catch (e) {
        console.error('Error loading state:', e);
    }
}

// Save state to localStorage
function saveState() {
    try {
        localStorage.setItem('chatMessages', JSON.stringify(messages));
        localStorage.setItem('selectedModel', selectedModel);
    } catch (e) {
        console.error('Error saving state:', e);
    }
}

// Update UI
function updateUI() {
    // Update model display
    currentModelName.textContent = MODEL_NAMES[selectedModel] || selectedModel;
    messageCount.textContent = messages.length;

    // Show/hide welcome message
    if (messages.length === 0) {
        welcomeMessage.style.display = 'block';
        messagesContainer.innerHTML = '';
    } else {
        welcomeMessage.style.display = 'none';
        renderMessages();
    }
}

// Render all messages
function renderMessages() {
    messagesContainer.innerHTML = messages.map((msg, index) => createMessageHTML(msg, index)).join('');
    scrollToBottom();
}

// Create HTML for a message
function createMessageHTML(message, index) {
    const isUser = message.role === 'user';
    const avatarText = isUser ? 'You' : 'AI';
    const formattedContent = formatMessage(message.content);
    const meta = message.model ? `<div class="message-meta">Model: ${MODEL_NAMES[message.model] || message.model}</div>` : '';

    return `
        <div class="message ${message.role}">
            <div class="message-avatar">${avatarText}</div>
            <div class="message-content">
                ${formattedContent}
                ${meta}
            </div>
        </div>
    `;
}

// Format message content (basic markdown-like formatting)
function formatMessage(content) {
    if (!content) return '';

    // Ensure content is a string
    if (typeof content !== 'string') {
        content = JSON.stringify(content);
    }

    // Escape HTML
    let formatted = content
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');

    // Code blocks
    formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, (match, lang, code) => {
        return `<pre><code>${code.trim()}</code></pre>`;
    });

    // Inline code
    formatted = formatted.replace(/`([^`]+)`/g, '<code>$1</code>');

    // Bold
    formatted = formatted.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');

    // Italic
    formatted = formatted.replace(/\*([^*]+)\*/g, '<em>$1</em>');

    // Line breaks
    formatted = formatted.replace(/\n/g, '<br>');

    // Wrap in paragraphs
    return `<p>${formatted}</p>`;
}

// Send message
async function sendMessage(event) {
    if (event) event.preventDefault();

    const content = messageInput.value.trim();
    if (!content || isLoading) return;

    // Clear input
    messageInput.value = '';
    messageInput.style.height = 'auto';
    sendBtn.disabled = true;

    // Add user message
    const userMessage = { role: 'user', content };
    messages.push(userMessage);
    updateUI();

    // Show loading
    isLoading = true;
    showLoading();

    try {
        console.log('Sending request to:', API_URL);
        console.log('Request body:', { message: content, model: selectedModel });

        const response = await fetch(API_URL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: content,
                model: selectedModel
            })
        });

        console.log('Response status:', response.status);
        console.log('Response ok:', response.ok);

        const data = await response.json();
        console.log('Response data:', data);

        hideLoading();

        // API returns {success: true, data: {message: "...", model: "...", query: "..."}}
        if (data.success && data.data && data.data.message) {
            let messageContent = data.data.message;

            // Clean up Llama special tokens if present
            if (typeof messageContent === 'string') {
                messageContent = messageContent.replace(/<\|start_header_id\|>assistant<\|end_header_id\|>\n*/g, '');
            }

            const assistantMessage = {
                role: 'assistant',
                content: messageContent,
                model: data.data.model || selectedModel
            };
            messages.push(assistantMessage);
        } else {
            const errorMessage = {
                role: 'assistant',
                content: data.error?.message || data.data?.message || 'Sorry, I encountered an error. Please try again.',
                model: selectedModel,
                error: true
            };
            messages.push(errorMessage);
        }
    } catch (error) {
        hideLoading();
        console.error('Fetch error:', error);
        console.error('Error name:', error.name);
        console.error('Error message:', error.message);

        let errorContent = 'Unable to connect to the server. Please make sure the kdeps agent is running.';
        if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
            errorContent = 'Network error: Could not reach the API server. Check if kdeps is running.';
        } else if (error.name === 'SyntaxError') {
            errorContent = 'Server returned invalid response. The LLM might still be processing.';
        } else {
            errorContent = `Error: ${error.message}`;
        }

        const errorMessage = {
            role: 'assistant',
            content: errorContent,
            model: selectedModel,
            error: true
        };
        messages.push(errorMessage);
    }

    isLoading = false;
    saveState();
    updateUI();
}

// Show loading indicator
function showLoading() {
    const loadingHTML = `
        <div class="message assistant" id="loading-message">
            <div class="message-avatar">AI</div>
            <div class="message-content">
                <div class="loading">
                    <div class="loading-dot"></div>
                    <div class="loading-dot"></div>
                    <div class="loading-dot"></div>
                </div>
            </div>
        </div>
    `;
    messagesContainer.insertAdjacentHTML('beforeend', loadingHTML);
    scrollToBottom();
}

// Hide loading indicator
function hideLoading() {
    const loadingMessage = document.getElementById('loading-message');
    if (loadingMessage) {
        loadingMessage.remove();
    }
}

// Send suggestion
function sendSuggestion(text) {
    messageInput.value = text;
    sendBtn.disabled = false;
    sendMessage();
}

// Clear chat
function clearChat() {
    messages = [];
    saveState();
    updateUI();
}

// Update model
function updateModel() {
    selectedModel = modelSelect.value;
    currentModelName.textContent = MODEL_NAMES[selectedModel] || selectedModel;
    saveState();
}

// Handle keyboard input
function handleKeyDown(event) {
    // Send on Enter (without Shift)
    if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault();
        if (!sendBtn.disabled) {
            sendMessage();
        }
    }
}

// Auto-resize textarea
function autoResize(textarea) {
    textarea.style.height = 'auto';
    textarea.style.height = Math.min(textarea.scrollHeight, 200) + 'px';
}

// Scroll to bottom of chat
function scrollToBottom() {
    chatContainer.scrollTop = chatContainer.scrollHeight;
}
