import React, { useEffect, useState } from "react";
import axios from "axios";

function App() {
  const [items, setItems] = useState([]);
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    // Fetch initial list of items
    axios
      .get("http://localhost:8080/api/items")
      .then((response) => {
        setItems(response.data);
      })
      .catch((error) => {
        console.error("There was an error fetching the items!", error);
      });
  }, []);

  const handleAddItem = (e) => {
    e.preventDefault();

    const itemData = {
      name,
      price: parseFloat(price),
    };

    // Post new item to backend
    axios
      .post("http://localhost:8080/api/items/add", itemData)
      .then((response) => {
        setMessage("Item added successfully!");
        setItems([
          ...items,
          { id: response.data.id, name, price: itemData.price },
        ]);
        setName("");
        setPrice("");
      })
      .catch((error) => {
        console.error("There was an error adding the item!", error);
        setMessage("Failed to add item.");
      });
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>Welcome to My App</h1>
        <p>This is the frontend powered by ReactJS.</p>

        <h2>Items List:</h2>
        <ul>
          {items.map((item) => (
            <li key={item.id}>
              {item.name} - ${item.price}
            </li>
          ))}
        </ul>

        <h2>Add New Item:</h2>
        <form onSubmit={handleAddItem}>
          <div>
            <label>
              Name:
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </label>
          </div>
          <div>
            <label>
              Price:
              <input
                type="number"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
                required
              />
            </label>
          </div>
          <button type="submit">Add Item</button>
        </form>

        {message && <p>{message}</p>}
      </header>
    </div>
  );
}

export default App;
