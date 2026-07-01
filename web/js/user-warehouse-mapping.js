let users = [];
let warehouses = [];
let assigned = [];

const userSelect = document.getElementById("userSelect");
const userSearch = document.getElementById("userSearch");

const warehouseSearch = document.getElementById("warehouseSearch");

const warehouseList = document.getElementById("warehouseList");

const saveBtn = document.getElementById("saveBtn");

async function loadUsers() {

    users = await API.users();

    renderUsers(users);

}

function renderUsers(list){

    userSelect.innerHTML =
        '<option value="">Select User</option>';

    list.forEach(u=>{

        const opt=document.createElement("option");

        opt.value=u.id;
        opt.textContent=u.email;

        userSelect.appendChild(opt);

    });

}

async function loadWarehouses(){

    warehouses = await API.warehouses();

    renderWarehouses();

}

function renderWarehouses(){

    warehouseList.innerHTML="";

    const filter =
        warehouseSearch.value.toLowerCase();

    warehouses
    .filter(w=>
        w.name.toLowerCase().includes(filter)
        ||
        w.id.toLowerCase().includes(filter)
    )
    .forEach(w=>{

        const div=document.createElement("div");

        div.className="checkbox-row";

        div.innerHTML=`
<label>

<input
type="checkbox"
value="${w.id}"
${assigned.includes(w.id)?"checked":""}
>

${w.id} - ${w.name}

</label>
`;

        warehouseList.appendChild(div);

    });

}

async function loadAssignments(){

    if(!userSelect.value){

        assigned=[];

        renderWarehouses();

        saveBtn.disabled=true;

        return;

    }

    saveBtn.disabled=false;

    const data =
        await API.json(
            `/api/admin/users/${userSelect.value}/warehouses`
        );

    assigned=data.warehouse_ids||[];

    renderWarehouses();

}

userSearch.oninput=()=>{

    const q=userSearch.value.toLowerCase();

    renderUsers(

        users.filter(u=>

            u.email
            .toLowerCase()
            .includes(q)

        )

    );

};

warehouseSearch.oninput=renderWarehouses;

userSelect.onchange=loadAssignments;

document
.getElementById("selectAllBtn")
.onclick=()=>{

assigned=warehouses.map(w=>w.id);

renderWarehouses();

};

document
.getElementById("clearAllBtn")
.onclick=()=>{

assigned=[];

renderWarehouses();

};

saveBtn.onclick=async()=>{

const ids=[

...warehouseList.querySelectorAll(
"input[type=checkbox]:checked"
)

].map(c=>c.value);

try{

await API.json(

`/api/admin/users/${userSelect.value}/warehouses`,

{

method:"PUT",

body:JSON.stringify({

warehouse_ids:ids

})

}

);

showToast("Mapping updated");

}
catch(e){

showToast(e.message,true);

}

};

loadUsers();

loadWarehouses();