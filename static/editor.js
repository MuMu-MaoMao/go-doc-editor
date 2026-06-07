// ---------- 前端逻辑 ----------
import { getToken, removeToken, isLoggedIn, redirectToLogin } from '/static/auth.js';

if (!isLoggedIn()) {
    redirectToLogin();
}

let currentFile = null;
let fileList = [];
let categories = [];     // 扁平分类列表
let selectedCategoryId = null;  // null=全部, '__uncat__'=未分类

const fileListEl = document.getElementById('fileList');
const editor = document.getElementById('editor');
const currentFileSpan = document.getElementById('currentFile');
const statusSpan = document.getElementById('statusMsg');
const refreshBtn = document.getElementById('refreshBtn');
const createBtn = document.getElementById('createBtn');
const saveBtn = document.getElementById('saveBtn');
const deleteCurrentBtn = document.getElementById('deleteCurrentBtn');
const newFileNameInput = document.getElementById('newFileName');
const categoryTreeEl = document.getElementById('categoryTree');
const newCategoryBtn = document.getElementById('newCategoryBtn');
const newCategoryForm = document.getElementById('newCategoryForm');
const newCatName = document.getElementById('newCatName');
const newCatParent = document.getElementById('newCatParent');
const newCatConfirm = document.getElementById('newCatConfirm');
const fileCategorySelect = document.getElementById('fileCategorySelect');

// ==============================
// 工具函数
// ==============================

function setStatus(msg, isError = false) {
    statusSpan.textContent = msg;
    statusSpan.style.color = isError ? '#b91c1c' : '#555555';
    setTimeout(() => {
        if (statusSpan.textContent === msg && !isError) statusSpan.style.color = '#555555';
    }, 2500);
}

function authHeaders(extra = {}) {
    return { ...extra, 'Authorization': `Bearer ${getToken()}` };
}

async function checkAuthResponse(response) {
    if (response.status === 401) {
        removeToken();
        try { const data = await response.json(); console.error('认证失败:', data.error); }
        catch (e) { console.error('认证失败: token 已过期'); }
        redirectToLogin();
        return false;
    }
    return true;
}

// ==============================
// 分类管理
// ==============================

function buildCategoryTree(flatCats) {
    const map = {};
    const roots = [];
    flatCats.forEach(c => { map[c.id] = { ...c, children: [] }; });
    flatCats.forEach(c => {
        const node = map[c.id];
        if (c.parentId === null) {
            roots.push(node);
        } else if (map[c.parentId]) {
            map[c.parentId].children.push(node);
        }
    });
    return roots;
}

// 获取分类层级深度（用于 CSS 缩进）
function getCategoryDepth(flatCats, catId) {
    let depth = 0;
    let cat = flatCats.find(c => c.id === catId);
    while (cat && cat.parentId !== null) {
        depth++;
        cat = flatCats.find(c => c.id === cat.parentId);
    }
    return depth;
}

// 分类折叠状态（存分类ID，展开状态的父分类不在其中）
let collapsedCategories = new Set();

// 判断一个分类是否有子分类
function hasChildren(catId) {
    return categories.some(c => c.parentId === catId);
}

function renderCategoryTree() {
    categoryTreeEl.innerHTML = `
        <li class="cat-all${selectedCategoryId === null ? ' active' : ''}" data-cat-id="">全部文档</li>
        <li class="cat-uncat${selectedCategoryId === '__uncat__' ? ' active' : ''}" data-cat-id="__uncat__">未分类</li>
    `;

    function renderChildren(parentId, depth) {
        const children = categories.filter(c => {
            if (parentId === null) return c.parentId === null;
            return c.parentId === parentId;
        });
        children.forEach(cat => {
            const hasKids = hasChildren(cat.id);
            const isCollapsed = collapsedCategories.has(cat.id);
            const depthClass = depth === 0 ? '' : depth === 1 ? 'cat-child' : 'cat-grandchild';
            const li = document.createElement('li');
            li.className = `${depthClass}${selectedCategoryId === cat.id ? ' active' : ''}`;
            li.dataset.catId = cat.id;

            // 展开/折叠箭头 + 名称
            const toggleSpan = document.createElement('span');
            toggleSpan.className = 'cat-toggle';
            toggleSpan.textContent = hasKids ? (isCollapsed ? '▶' : '▼') : '  ';
            toggleSpan.style.cssText = 'display:inline-block;width:14px;font-size:0.65rem;cursor:pointer;flex-shrink:0;';

            const nameSpan = document.createElement('span');
            nameSpan.textContent = cat.name;
            nameSpan.style.cssText = 'flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;';

            const actions = document.createElement('span');
            actions.className = 'cat-actions';
            actions.innerHTML = `
                <span class="cat-rename" title="重命名">✏️</span>
                <span class="cat-delete" title="删除">✖</span>
            `;

            li.style.cssText = 'gap:2px;';
            li.appendChild(toggleSpan);
            li.appendChild(nameSpan);
            li.appendChild(actions);

            // 点击箭头：切换折叠
            toggleSpan.addEventListener('click', (e) => {
                e.stopPropagation();
                if (isCollapsed) {
                    collapsedCategories.delete(cat.id);
                } else {
                    collapsedCategories.add(cat.id);
                }
                renderCategoryTree();
            });

            // 点击名称或空白区域：选中分类
            nameSpan.addEventListener('click', (e) => {
                e.stopPropagation();
                selectCategory(cat.id);
            });

            // 重命名
            li.querySelector('.cat-rename')?.addEventListener('click', (e) => {
                e.stopPropagation();
                const newName = prompt('输入新名称：', cat.name);
                if (newName && newName.trim() && newName !== cat.name) {
                    renameCategory(cat.id, newName.trim());
                }
            });
            // 删除
            li.querySelector('.cat-delete')?.addEventListener('click', (e) => {
                e.stopPropagation();
                if (confirm(`确定删除分类 "${cat.name}" 吗？`)) {
                    deleteCategory(cat.id);
                }
            });

            categoryTreeEl.appendChild(li);

            // 折叠状态下不渲染子分类
            if (!isCollapsed) {
                renderChildren(cat.id, depth + 1);
            }
        });
    }
    renderChildren(null, 0);
}

function selectCategory(catId) {
    if (selectedCategoryId === catId) return;
    selectedCategoryId = catId;
    renderCategoryTree();
    loadFilesWithFilter();
}

// 分类树点击事件（委托）
categoryTreeEl.addEventListener('click', (e) => {
    const li = e.target.closest('li');
    if (!li) return;
    if (e.target.closest('.cat-actions')) return;
    const catId = li.dataset.catId;
    if (catId === '') {
        selectedCategoryId = null;
    } else if (catId === '__uncat__') {
        selectedCategoryId = '__uncat__';
    } else {
        selectedCategoryId = parseInt(catId, 10);
    }
    renderCategoryTree();
    loadFilesWithFilter();
});

async function fetchCategories() {
    try {
        const res = await fetch('/api/categories', { headers: authHeaders() });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success && data.tree) {
            // 展平树为数组便于处理
            categories = [];
            function flatten(tree) {
                tree.forEach(c => {
                    const children = c.children || [];
                    delete c.children;
                    categories.push(c);
                    flatten(children);
                });
            }
            flatten(data.tree);
            // 重建父分类下拉选项
            rebuildCategorySelects();
        }
    } catch (err) {
        console.error('加载分类失败:', err);
    }
}

// 重建所有分类下拉框（文件分类选择器 + 新建分类表单）
function getCategoryPath(flatCats, catId) {
    const names = [];
    let cat = flatCats.find(c => c.id === catId);
    while (cat) {
        names.unshift(cat.name);
        cat = cat.parentId ? flatCats.find(c => c.id === cat.parentId) : null;
    }
    return names.join(' / ');
}

function rebuildCategorySelects() {
    // 文件分类选择器（编辑器工具栏）
    fileCategorySelect.innerHTML = '<option value="">未分类</option>';
    // 新建分类的父分类选择器
    newCatParent.innerHTML = '<option value="">作为大类</option>';

    const roots = buildCategoryTree(categories);
    function addOptions(nodes, prefix, target) {
        nodes.forEach(n => {
            const opt = document.createElement('option');
            opt.value = n.id;
            opt.textContent = prefix + getCategoryPath(categories, n.id);
            target.appendChild(opt);
            addOptions(n.children, prefix + '  ', target);
        });
    }
    addOptions(roots, '', fileCategorySelect);
    addOptions(roots, '', newCatParent);
}

async function createCategory(name, parentId) {
    const body = { name };
    if (parentId) body.parentId = parseInt(parentId, 10);
    try {
        const res = await fetch('/api/categories', {
            method: 'POST',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify(body),
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`✅ 分类 "${name}" 已创建`);
            await fetchCategories();
            renderCategoryTree();
        } else {
            setStatus('创建分类失败: ' + (data.error || ''), true);
        }
    } catch (err) {
        setStatus('创建分类失败: ' + err.message, true);
    }
}

async function renameCategory(catId, newName) {
    try {
        const res = await fetch(`/api/categories/${catId}`, {
            method: 'PUT',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ name: newName }),
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`✅ 已重命名`);
            await fetchCategories();
            renderCategoryTree();
            // 如果当前打开的文件受影响，刷新其分类选择
            if (currentFile) loadCurrentFileCategory();
        } else {
            setStatus('重命名失败: ' + (data.error || ''), true);
        }
    } catch (err) {
        setStatus('重命名失败: ' + err.message, true);
    }
}

async function deleteCategory(catId) {
    try {
        const res = await fetch(`/api/categories/${catId}`, {
            method: 'DELETE',
            headers: authHeaders(),
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`✅ 分类已删除`);
            if (selectedCategoryId === catId) {
                selectedCategoryId = null;
            }
            await fetchCategories();
            renderCategoryTree();
            loadFilesWithFilter();
        } else {
            setStatus('删除失败: ' + (data.error || ''), true);
        }
    } catch (err) {
        setStatus('删除失败: ' + err.message, true);
    }
}

// 新建分类表单
newCategoryBtn?.addEventListener('click', async () => {
    const isHidden = !newCategoryForm.classList.contains('visible');
    newCategoryForm.classList.toggle('visible');
    if (isHidden) {
        newCatName.value = '';
        // 从服务器刷新分类数据，确保下拉框有最新选项
        await fetchCategories();
        rebuildCategorySelects();
    }
});

newCatConfirm?.addEventListener('click', () => {
    const name = newCatName.value.trim();
    if (!name) { setStatus('请输入分类名称', true); return; }
    const parentId = newCatParent.value;
    createCategory(name, parentId || null);
    newCategoryForm.classList.remove('visible');
});

newCatName?.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') newCatConfirm?.click();
});

// ==============================
// 文档-分类关联
// ==============================

// 加载文件时按分类筛选
async function loadFilesWithFilter() {
    try {
        let url = '/api/files';
        if (selectedCategoryId && selectedCategoryId !== '__uncat__') {
            url = `/api/files?category=${selectedCategoryId}`;
        }
        const res = await fetch(url, { headers: authHeaders() });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            fileList = data.files || [];
            // 除非分类是"未分类"，否则过滤掉未分类文件
            if (selectedCategoryId === '__uncat__') {
                // 特殊处理：后端需要返回未分类文件，目前用单独的端点
                const uncatRes = await fetch('/api/files?uncategorized=1', { headers: authHeaders() });
                if (await checkAuthResponse(uncatRes)) {
                    const uncatData = await uncatRes.json();
                    if (uncatData.success) fileList = uncatData.files || [];
                }
            }
            renderFileList();
        } else {
            throw new Error(data.error || '未知错误');
        }
    } catch (err) {
        setStatus('加载文件失败: ' + err.message, true);
    }
}

// 设置当前文件的分类
async function setFileCategory(filename, categoryId) {
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(filename)}/category`, {
            method: 'PUT',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ categoryId: categoryId || null }),
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`✅ 分类已更新`);
        } else {
            setStatus('设置分类失败: ' + (data.error || ''), true);
        }
    } catch (err) {
        setStatus('设置分类失败: ' + err.message, true);
    }
}

// 当前文件分类选择变更
fileCategorySelect?.addEventListener('change', () => {
    if (!currentFile) return;
    const val = fileCategorySelect.value;
    const catId = val ? parseInt(val, 10) : null;
    setFileCategory(currentFile, catId);
});

// 加载当前文件的分类
async function loadCurrentFileCategory() {
    if (!currentFile) return;
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(currentFile)}/category`, {
            headers: authHeaders(),
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            fileCategorySelect.value = data.category ? String(data.category.id) : '';
        }
    } catch (err) {
        console.error('加载文件分类失败:', err);
    }
}

// ==============================
// 文档 CRUD（原有 + 分类适配）
// ==============================

async function fetchFileList() {
    try {
        const res = await fetch('/api/files', { headers: authHeaders() });
        if (!await checkAuthResponse(res)) return [];
        const data = await res.json();
        if (data.success) {
            fileList = data.files || [];
            renderFileList();
            return fileList;
        } else {
            throw new Error(data.error || '未知错误');
        }
    } catch (err) {
        setStatus('加载文件列表失败: ' + err.message, true);
        fileListEl.innerHTML = '<li style="justify-content:center;">加载失败，请刷新</li>';
        return [];
    }
}

function renderFileList() {
    if (!fileList.length) {
        fileListEl.innerHTML = '<li style="justify-content:center;">📭 暂无文档</li>';
        return;
    }
    const fragment = document.createDocumentFragment();
    fileList.forEach(filename => {
        const li = document.createElement('li');
        li.className = (currentFile === filename) ? 'active' : '';
        const nameSpan = document.createElement('span');
        nameSpan.className = 'file-name';
        nameSpan.textContent = filename;
        const delBtn = document.createElement('button');
        delBtn.textContent = '✖';
        delBtn.className = 'del-btn';
        delBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            deleteFile(filename);
        });
        li.appendChild(nameSpan);
        li.appendChild(delBtn);
        li.addEventListener('click', () => openFile(filename));
        fragment.appendChild(li);
    });
    fileListEl.innerHTML = '';
    fileListEl.appendChild(fragment);
}

async function openFile(filename) {
    if (!filename) return;
    setStatus(`正在加载 ${filename} ...`);
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(filename)}`, { headers: authHeaders() });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            editor.value = data.content;
            currentFile = filename;
            currentFileSpan.textContent = `${filename}`;
            setStatus(`已打开 ${filename}`);
            renderFileList();
            loadCurrentFileCategory();
        } else {
            throw new Error(data.error || '读取失败');
        }
    } catch (err) {
        setStatus(`打开失败: ${err.message}`, true);
        if (err.message.includes('不存在')) {
            if (currentFile === filename) {
                editor.value = '';
                currentFile = null;
                currentFileSpan.textContent = '未选择文件';
                fileCategorySelect.style.display = 'none';
            }
            loadFilesWithFilter();
        }
    }
}

async function saveCurrent() {
    if (!currentFile) {
        setStatus('请先选择或新建一个文档', true);
        return;
    }
    const content = editor.value;
    setStatus(`保存中...`);
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(currentFile)}`, {
            method: 'POST',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ content })
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`✅ 已保存 ${currentFile}`);
            loadFilesWithFilter();
        } else {
            throw new Error(data.error || '保存失败');
        }
    } catch (err) {
        setStatus(`保存失败: ${err.message}`, true);
    }
}

async function deleteFile(filename) {
    if (!confirm(`确定删除 "${filename}" 吗？不可恢复。`)) return;
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(filename)}`, {
            method: 'DELETE',
            headers: authHeaders()
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`已删除 ${filename}`);
            if (currentFile === filename) {
                editor.value = '';
                currentFile = null;
                currentFileSpan.textContent = '未选择文件';
                fileCategorySelect.style.display = 'none';
            }
            loadFilesWithFilter();
        } else {
            throw new Error(data.error || '删除失败');
        }
    } catch (err) {
        setStatus(`删除失败: ${err.message}`, true);
    }
}

async function deleteCurrent() {
    if (!currentFile) {
        setStatus('没有打开任何文件', true);
        return;
    }
    await deleteFile(currentFile);
}

async function createDocument() {
    let newName = newFileNameInput.value.trim();
    if (!newName) {
        setStatus('请输入文件名', true);
        return;
    }
    if (newName.includes('/') || newName.includes('\\') || newName.includes('..')) {
        setStatus('文件名不能包含 / \\ 或 ..', true);
        return;
    }
    if (fileList.includes(newName) && !confirm(`文件 ${newName} 已存在，是否覆盖？`)) {
        return;
    }
    setStatus(`正在创建 ${newName} ...`);
    try {
        const res = await fetch(`/api/file/${encodeURIComponent(newName)}`, {
            method: 'POST',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ content: '' })
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus(`创建成功，正在打开...`);
            newFileNameInput.value = '';
            await loadFilesWithFilter();
            await openFile(newName);
        } else {
            throw new Error(data.error || '创建失败');
        }
    } catch (err) {
        setStatus(`创建失败: ${err.message}`, true);
    }
}

// ==============================
// 事件绑定
// ==============================

refreshBtn.addEventListener('click', () => loadFilesWithFilter().then(() => setStatus('列表已刷新')));
createBtn.addEventListener('click', createDocument);
saveBtn.addEventListener('click', saveCurrent);
deleteCurrentBtn.addEventListener('click', deleteCurrent);
newFileNameInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') createDocument();
});

// ====== 标注工具栏按钮 ======
document.getElementById('addAnnotationBtn')?.addEventListener('click', () => {
    if (!currentFile) {
        setStatus('请先打开一个文档', true);
        return;
    }
    // 优先使用选中的文本
    const sel = getTextareaSelection();
    if (sel) {
        annoHighlightBtn?.click();
        return;
    }
    // 无选中文本时手动输入
    const text = prompt('输入要标注的文本：');
    if (!text) return;
    const targetFile = prompt('关联到哪个文档？（可选）：', '');
    if (targetFile === null) return;
    const comment = prompt('添加评语（可选）：', '');
    if (comment === null) return;

    // 在文档中查找文本位置
    const startPos = editor.value.indexOf(text);
    if (startPos === -1) {
        setStatus('⚠️ 在文档中未找到该文本', true);
        return;
    }
    const endPos = startPos + text.length;

    (async () => {
        const body = {
            sourceFilename: currentFile,
            selectedText: text,
            positionStart: startPos,
            positionEnd: endPos,
        };
        if (targetFile.trim()) body.targetFilename = targetFile.trim();
        if (comment.trim()) body.comment = comment.trim();

        try {
            const res = await fetch('/api/annotations', {
                method: 'POST',
                headers: authHeaders({ 'Content-Type': 'application/json' }),
                body: JSON.stringify(body),
            });
            if (!await checkAuthResponse(res)) return;
            const data = await res.json();
            if (data.success) {
                setStatus('✅ 标注已创建');
                await loadAnnotations();
            } else {
                setStatus('创建标注失败: ' + (data.error || ''), true);
            }
        } catch (err) {
            setStatus('创建标注失败: ' + err.message, true);
        }
    })();
});

// ====== AI 集成 ======
document.getElementById('aiSendToChatBtn')?.addEventListener('click', () => {
    if (!editor.value) {
        setStatus('⚠️ 编辑器为空，无法附带文档', true);
        return;
    }
    const panel = document.getElementById('aiChatPanel');
    const toggle = document.getElementById('aiChatToggle');
    if (panel && toggle && !panel.classList.contains('visible')) {
        toggle.click();
    }
    const attachBtn = document.getElementById('aiChatAttachDoc');
    if (attachBtn && !attachBtn.classList.contains('active')) {
        attachBtn.click();
    }
    setStatus('AI 面板已打开，附带当前文档');
});

// ====== 全局最小化/展开 + 按钮拖拽 ======
const appContainer = document.querySelector('.app');
const globalMinimizeBtn = document.getElementById('minimizeBtn');
const globalRestoreBtn = document.getElementById('globalRestoreBtn');
const aiToggleBtn = document.getElementById('aiChatToggle');

// 按钮原始位置记录
const btnOrigins = new Map();

// 初始化按钮位置（right/bottom → left/top）
function initDraggablePositions() {
    const now = Date.now();
    [globalRestoreBtn, aiToggleBtn].forEach(btn => {
        if (!btn || btn._initialized) return;
        const rect = btn.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) return;
        btn.style.left = rect.left + 'px';
        btn.style.top = rect.top + 'px';
        btn.style.right = 'auto';
        btn.style.bottom = 'auto';
        btnOrigins.set(btn, { left: rect.left, top: rect.top });
        btn._initialized = true;
    });
}

// 拖拽状态
let dragState = null;
let didDrag = false; // 标记是否真正拖拽过（防止拖拽后触发点击）

function onPointerDown(e) {
    const btn = e.currentTarget;
    if (!appContainer.classList.contains('minimized')) return;
    const rect = btn.getBoundingClientRect();
    dragState = {
        btn: btn,
        offsetX: e.clientX - rect.left,
        offsetY: e.clientY - rect.top,
    };
    didDrag = false;
    btn.classList.add('dragging');
    e.preventDefault();
}

function onPointerMove(e) {
    if (!dragState) return;
    const { btn, offsetX, offsetY } = dragState;
    const dx = e.clientX - offsetX - parseFloat(btn.style.left || 0);
    const dy = e.clientY - offsetY - parseFloat(btn.style.top || 0);
    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) didDrag = true;
    btn.style.left = (e.clientX - offsetX) + 'px';
    btn.style.top = (e.clientY - offsetY) + 'px';
}

function onPointerUp(e) {
    if (!dragState) return;
    dragState.btn.classList.remove('dragging');
    dragState = null;
    // 拖拽结束后保存按钮当前位置（下次最小化时用）
    saveDragPositions();
}

// ====== 切换最小化/展开 ======

// 保存按钮的拖拽后位置（最小化时停留的位置）
let dragPositions = null;

// 保存当前拖拽后的位置（每次拖拽更新）
function saveDragPositions() {
    dragPositions = {};
    [globalRestoreBtn, aiToggleBtn].forEach(btn => {
        if (!btn) return;
        const rect = btn.getBoundingClientRect();
        dragPositions[btn.id] = { left: rect.left, top: rect.top };
    });
}

// 最小化 app
function minimizeApp() {
    // 冻结按钮当前位置
    [globalRestoreBtn, aiToggleBtn].forEach(btn => {
        if (!btn) return;
        const rect = btn.getBoundingClientRect();
        if (rect.width > 0) {
            btn.style.left = rect.left + 'px';
            btn.style.top = rect.top + 'px';
        }
        btn._initialized = true;
        btn.style.right = 'auto';
        btn.style.bottom = 'auto';
        // 记录原点（仅首次）
        if (!btnOrigins.has(btn)) {
            btnOrigins.set(btn, { left: rect.left, top: rect.top });
        }
    });
    if (dragPositions) {
        // 如果有之前拖拽的位置，立刻跳到拖拽位置（不做过渡）
        [globalRestoreBtn, aiToggleBtn].forEach(btn => {
            if (!btn || !dragPositions[btn.id]) return;
            btn.style.transition = 'none';
            btn.style.left = dragPositions[btn.id].left + 'px';
            btn.style.top = dragPositions[btn.id].top + 'px';
            // 重新启用过渡
            requestAnimationFrame(() => {
                btn.style.transition = '';
            });
        });
    }

    appContainer.classList.add('minimized');
}

// 展开 app
function restoreApp() {
    // 展开按钮飞回原点
    const ro = btnOrigins.get(globalRestoreBtn);
    if (ro) { globalRestoreBtn.style.left = ro.left + 'px'; globalRestoreBtn.style.top = ro.top + 'px'; }
    // AI 按钮飞回原点
    const ao = btnOrigins.get(aiToggleBtn);
    if (ao) { aiToggleBtn.style.left = ao.left + 'px'; aiToggleBtn.style.top = ao.top + 'px'; }

    // app 淡入
    setTimeout(() => { appContainer.classList.remove('minimized'); }, 160);
}

// —— 工具栏「—」按钮：最小化 ——
globalMinimizeBtn?.addEventListener('click', minimizeApp);

// —— ⚫ 展开/收起切换按钮 ——
globalRestoreBtn?.addEventListener('click', (e) => {
    if (didDrag) {
        didDrag = false;
        e.stopImmediatePropagation();
        return;
    }
    if (appContainer.classList.contains('minimized')) {
        restoreApp();
    } else {
        minimizeApp();
    }
});

// —— 全局事件 ——
document.addEventListener('mousemove', onPointerMove);
document.addEventListener('mouseup', onPointerUp);
[globalRestoreBtn, aiToggleBtn].forEach(btn => {
    if (!btn) return;
    // 拖拽开始
    btn.addEventListener('mousedown', onPointerDown);
    // 拖拽后释放产生的 click 一律拦截（防止意外触发功能）
    btn.addEventListener('click', (e) => {
        if (didDrag) {
            didDrag = false;
            e.stopImmediatePropagation();
            e.preventDefault();
        }
    }, true); // 捕获阶段拦截，确保先于其他 handler
});

// —— 初始化位置（确保 DOM 渲染后） ——
if (document.readyState === 'complete') {
    initDraggablePositions();
} else {
    window.addEventListener('load', initDraggablePositions);
}

// ==============================
// 标注（Annotation）功能
// ==============================

const annotationPopup = document.getElementById('annotationPopup');
const annoHighlightBtn = document.getElementById('annoHighlightBtn');
const annotationSection = document.getElementById('annotationSection');
const annotationListEl = document.getElementById('annotationList');
const referenceSection = document.getElementById('referenceSection');
const referenceListEl = document.getElementById('referenceList');

// 获取 textarea 中选中文本和位置
function getTextareaSelection() {
    if (!currentFile) return null;
    const start = editor.selectionStart;
    const end = editor.selectionEnd;
    if (start === end) return null;
    const text = editor.value.substring(start, end).trim();
    if (!text) return null;
    return { text, start, end };
}

// 计算 textarea 中选中区域的屏幕位置（近似）
function getTextareaSelectionPosition() {
    const start = editor.selectionStart;
    const text = editor.value.substring(0, start);
    const lines = text.split('\n');
    const lineNum = lines.length;
    const colNum = lines[lines.length - 1].length;

    // 估算字符宽度和行高
    const charWidth = 8.5;
    const lineHeight = 22;
    const editorRect = editor.getBoundingClientRect();
    const editorStyle = window.getComputedStyle(editor);
    const paddingLeft = parseFloat(editorStyle.paddingLeft);
    const paddingTop = parseFloat(editorStyle.paddingTop);

    return {
        left: editorRect.left + paddingLeft + colNum * charWidth,
        top: editorRect.top + paddingTop + (lineNum - 1) * lineHeight
    };
}

// 选中文本时弹出标注按钮
editor?.addEventListener('mouseup', () => {
    const sel = getTextareaSelection();
    if (sel) {
        const pos = getTextareaSelectionPosition();
        annotationPopup.classList.add('visible');
        annotationPopup.style.left = pos.left + 'px';
        annotationPopup.style.top = (pos.top - 30) + 'px';
    } else {
        setTimeout(() => {
            if (!getTextareaSelection()) annotationPopup.classList.remove('visible');
        }, 200);
    }
});

editor?.addEventListener('mousedown', () => {
    annotationPopup.classList.remove('visible');
});

annoHighlightBtn?.addEventListener('click', async () => {
    const sel = getTextareaSelection();
    if (!sel) return;

    const targetFile = prompt('关联到哪个文档？（可选，留空仅标注不关联）：', '');
    if (targetFile === null) return; // 取消
    const comment = prompt('添加评语（可选）：', '');
    if (comment === null) return; // 取消

    try {
        const body = {
            sourceFilename: currentFile,
            selectedText: sel.text,
            positionStart: sel.start,
            positionEnd: sel.end,
        };
        if (targetFile.trim()) body.targetFilename = targetFile.trim();
        if (comment.trim()) body.comment = comment.trim();

        const res = await fetch('/api/annotations', {
            method: 'POST',
            headers: authHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify(body),
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            setStatus('✅ 标注已创建');
            annotationPopup.classList.remove('visible');
            await loadAnnotations();
        } else {
            setStatus('创建标注失败: ' + (data.error || ''), true);
        }
    } catch (err) {
        setStatus('创建标注失败: ' + err.message, true);
    }
});

async function loadAnnotations() {
    if (!currentFile) {
        annotationSection.style.display = 'none';
        return;
    }
    annotationSection.style.display = 'block';
    annotationListEl.innerHTML = '<li style="justify-content:center;font-size:0.8rem;color:#888;">加载中...</li>';

    try {
        // 加载标注
        const res = await fetch(`/api/file/${encodeURIComponent(currentFile)}/annotations`, {
            headers: authHeaders()
        });
        if (!await checkAuthResponse(res)) return;
        const data = await res.json();
        if (data.success) {
            const annos = data.annotations || [];
            if (annos.length === 0) {
                annotationListEl.innerHTML = '<li style="justify-content:center;font-size:0.8rem;color:#888;">暂无标注</li>';
            } else {
                annotationListEl.innerHTML = '';
                annos.forEach(a => {
                    const li = document.createElement('li');
                    li.className = 'annotation-item';
                    li.innerHTML = `
                        <div class="annotation-text">"${a.selectedText}"</div>
                        ${a.targetFilename ? `<div>→ <a href="#" class="annotation-link" data-file="${a.targetFilename}">${a.targetFilename}</a></div>` : ''}
                        ${a.comment ? `<div class="annotation-comment">💬 ${a.comment}</div>` : ''}
                        <span class="annotation-delete" data-id="${a.id}">删除</span>
                    `;
                    annotationListEl.appendChild(li);
                });
            }
        }

        // 加载引用
        const refRes = await fetch(`/api/file/${encodeURIComponent(currentFile)}/references`, {
            headers: authHeaders()
        });
        if (!await checkAuthResponse(refRes)) return;
        const refData = await refRes.json();
        if (refData.success) {
            const refs = refData.references || [];
            if (refs.length > 0) {
                referenceSection.style.display = '';
                referenceListEl.innerHTML = '';
                refs.forEach(r => {
                    const li = document.createElement('li');
                    li.className = 'reference-item';
                    li.innerHTML = `
                        <div>📎 <a href="#" class="annotation-link" data-file="${r.sourceFilename}">${r.sourceFilename}</a></div>
                        <div>"${r.selectedText.substring(0, 30)}${r.selectedText.length > 30 ? '...' : ''}"</div>
                    `;
                    referenceListEl.appendChild(li);
                });
            } else {
                referenceSection.style.display = 'none';
            }
        }
    } catch (err) {
        console.error('加载标注失败:', err);
    }
}

// 标注/引用中的文档链接点击
document.addEventListener('click', (e) => {
    const link = e.target.closest('.annotation-link');
    if (link) {
        e.preventDefault();
        const file = link.dataset.file;
        if (file) openFile(file);
    }
    const del = e.target.closest('.annotation-delete');
    if (del) {
        e.preventDefault();
        const id = del.dataset.id;
        if (confirm('确定删除此标注？')) {
            fetch(`/api/annotations/${id}`, {
                method: 'DELETE',
                headers: authHeaders(),
            }).then(r => r.json()).then(d => {
                if (d.success) {
                    setStatus('✅ 标注已删除');
                    loadAnnotations();
                } else {
                    setStatus('删除失败: ' + (d.error || ''), true);
                }
            });
        }
    }
});

// ==============================
// 初始化
// ==============================

async function init() {
    await fetchCategories();
    renderCategoryTree();
    rebuildCategorySelects();
    await fetchFileList();
    setStatus('就绪，点击文档开始编辑');
}

// 覆盖 openFile 以加载标注
const _origOpenFile = openFile;
openFile = async function(filename) {
    await _origOpenFile(filename);
    await loadAnnotations();
};

init();
