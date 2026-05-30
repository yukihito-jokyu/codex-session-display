import { useState } from 'react';
import logo from './assets/images/logo-universal.png';
import styles from './App.module.css';
import { Greet } from './wailsjs/go/main/App';
import { TitleBar } from './components/ui/TitleBar/TitleBar';

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const updateName = (e: any) => setName(e.target.value);
    const updateResultText = (result: string) => setResultText(result);

    function greet() {
        Greet(name).then(updateResultText);
    }

    return (
        <div className={styles.appContainer}>
            <TitleBar />
            <main className={styles.mainContent}>
                <img src={logo} className={styles.logo} alt="logo"/>
                <div className={styles.result}>{resultText}</div>
                <div className={styles.inputBox}>
                    <input className={styles.input} onChange={updateName} autoComplete="off" name="input" type="text"/>
                    <button className={styles.btn} onClick={greet}>Greet</button>
                </div>
            </main>
        </div>
    )
}

export default App;
