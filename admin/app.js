const API_BASE_URL = 'http://localhost:8080/api';
let isAdminLoggedIn = false;

document.addEventListener('DOMContentLoaded', () => {
    checkLogin();
    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
    document.getElementById('refreshBtn').addEventListener('click', loadPendingBenches);
    document.getElementById('logoutBtn').addEventListener('click', handleLogout);
}

function checkLogin() {
    const token = localStorage.getItem('adminToken');
    if (token) {
        isAdminLoggedIn = true;
        document.getElementById('loginSection').classList.add('hidden');
        document.getElementById('adminSection').classList.remove('hidden');
        loadPendingBenches();
    }
}

async function handleLogin(e) {
    e.preventDefault();
    
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    
    try {
        const response = await fetch(`${API_BASE_URL}/admin/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });
        
        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('adminToken', data.token);
            isAdminLoggedIn = true;
            document.getElementById('loginSection').classList.add('hidden');
            document.getElementById('adminSection').classList.remove('hidden');
            loadPendingBenches();
        } else {
            document.getElementById('loginError').textContent = 'Неверные учётные данные';
        }
    } catch (error) {
        console.error('Ошибка входа:', error);
        document.getElementById('loginError').textContent = 'Ошибка соединения с сервером';
    }
}

function handleLogout() {
    localStorage.removeItem('adminToken');
    isAdminLoggedIn = false;
    document.getElementById('loginSection').classList.remove('hidden');
    document.getElementById('adminSection').classList.add('hidden');
    document.getElementById('loginForm').reset();
    document.getElementById('loginError').textContent = '';
}

async function loadPendingBenches() {
    const container = document.getElementById('pendingList');
    container.innerHTML = '<p class="loading">Загрузка...</p>';
    
    try {
        const response = await fetch(`${API_BASE_URL}/admin/pending`);
        if (!response.ok) throw new Error('Ошибка загрузки');
        
        const benches = await response.json();
        
        if (benches.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <h3>✅ Все заявки обработаны!</h3>
                    <p>Нет ожидающих модерации скамеек</p>
                </div>
            `;
            return;
        }
        
        container.innerHTML = benches.map(bench => `
            <div class="pending-item" data-id="${bench.id}">
                <div class="bench-info">
                    <h3>${escapeHtml(bench.name)}</h3>
                    <p class="comment">${escapeHtml(bench.comment) || 'Без описания'}</p>
                    <p class="email">📧 ${escapeHtml(bench.email)}</p>
                    <p class="coords">📍 ${bench.latitude.toFixed(6)}, ${bench.longitude.toFixed(6)}</p>
                    ${bench.photo ? `<img src="${bench.photo}" alt="Фото" class="bench-photo">` : ''}
                </div>
                <div class="bench-actions">
                    <button class="btn btn-success" onclick="approveBench(${bench.id})">✔ Одобрить</button>
                    <button class="btn btn-warning" onclick="rejectBench(${bench.id})">✖ Отклонить</button>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Ошибка загрузки:', error);
        container.innerHTML = '<p class="error">Ошибка загрузки списка. Попробуйте позже.</p>';
    }
}

async function approveBench(id) {
    if (!confirm('Одобрить эту скамейку?')) return;
    
    try {
        const response = await fetch(`${API_BASE_URL}/admin/bench/${id}/approve`, {
            method: 'POST'
        });
        
        if (response.ok) {
            loadPendingBenches();
        } else {
            alert('Ошибка при одобрении');
        }
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка соединения с сервером');
    }
}

async function rejectBench(id) {
    if (!confirm('Отклонить эту заявку?')) return;
    
    try {
        const response = await fetch(`${API_BASE_URL}/admin/bench/${id}/reject`, {
            method: 'POST'
        });
        
        if (response.ok) {
            loadPendingBenches();
        } else {
            alert('Ошибка при отклонении');
        }
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка соединения с сервером');
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
