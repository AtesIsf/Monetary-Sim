import pandas as pd
import matplotlib.pyplot as plt
import os
from typing import Any

# Set file paths
DATA_DIR: str = os.path.dirname(os.path.abspath(__file__))
CSV_FILE: str = os.path.join(DATA_DIR, "sim.csv")


def calculate_macro_variables(df: pd.DataFrame) -> pd.DataFrame:
    """
    Aggregates micro-level agent data into macroeconomic time series.
    """
    # Group by Tick to get time series
    grouped = df.groupby('Tick')

    macro_data = pd.DataFrame(index=grouped.groups.keys())
    macro_data.index.name = 'Tick'

    # 1. Average Price Level - Mean of prices for all firms
    def calc_price_level(tick_df: pd.DataFrame) -> float:
        firms = tick_df[tick_df['Type'] == 'Firm']
        if firms.empty:
            return 0.0
        return float(firms['Price'].mean())

    macro_data['Average Price Level'] = grouped.apply(calc_price_level)

    # 2. Wealth Distribution (Gini Coefficient proxy / Concentration)
    # Let's look at total wealth held by Households vs Firms vs Banks
    macro_data['Household Wealth'] = (
        df[df['Type'] == 'Household'].groupby('Tick')['Balance'].sum()
    )
    macro_data['Firm Wealth'] = (
        df[df['Type'] == 'Firm'].groupby('Tick')['Balance'].sum()
    )
    macro_data['Bank Wealth'] = (
        df[df['Type'] == 'Bank'].groupby('Tick')['Balance'].sum()
    )

    # 3. Unemployment Rate
    # A household is unemployed if its Employer ID is its own ID
    def calc_unemployment(tick_df: pd.DataFrame) -> float:
        households = tick_df[tick_df['Type'] == 'Household']
        if households.empty:
            return 0.0
        unemployed = households[households['Employer'] == households['Id']]
        return float(len(unemployed) / len(households))

    macro_data['Unemployment Rate'] = grouped.apply(calc_unemployment)

    # 4. Total Employment (Absolute numbers)
    macro_data['Total Employed'] = grouped.apply(
        lambda d: len(
            d[(d['Type'] == 'Household') & (d['Employer'] != d['Id'])]
        )
    )

    # Calculate rolling averages for smoothing (window size 10)
    window = 10
    for col in macro_data.columns:
        macro_data[f'{col} Smooth'] = macro_data[col].rolling(window=window, center=True).mean()

    return macro_data


def plot_macro_variables(macro_df: pd.DataFrame) -> None:
    """
    Generates subplots for the calculated macroeconomic variables.
    """
    # Set up the figure and axes
    fig, axes = plt.subplots(nrows=2, ncols=2, figsize=(14, 10))
    fig.suptitle('Monetary Simulation: Macroeconomic Indicators', fontsize=16)

    # Plot 1: Average Price Level
    axes[0, 0].plot(
        macro_df.index, macro_df['Average Price Level'], color='black', alpha=0.3
    )
    axes[0, 0].plot(
        macro_df.index, macro_df['Average Price Level Smooth'], color='black', linewidth=2, label='Trend'
    )
    axes[0, 0].set_title('Average Price Level')
    axes[0, 0].set_xlabel('Tick')
    axes[0, 0].set_ylabel('Price')
    axes[0, 0].legend()
    axes[0, 0].grid(True)

    # Plot 2: Wealth Distribution
    axes[0, 1].plot(
        macro_df.index, macro_df['Household Wealth'],
        label='Households', color='blue', alpha=0.3
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Household Wealth Smooth'], color='blue', linewidth=2
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Firm Wealth'],
        label='Firms', color='green', alpha=0.3
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Firm Wealth Smooth'], color='green', linewidth=2
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Bank Wealth'],
        label='Bank', color='red', alpha=0.3
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Bank Wealth Smooth'], color='red', linewidth=2
    )
    axes[0, 1].set_title('Wealth Distribution by Sector')
    axes[0, 1].set_xlabel('Tick')
    axes[0, 1].set_ylabel('Total Balance')
    axes[0, 1].legend()
    axes[0, 1].grid(True)

    # Plot 3: Unemployment Rate
    axes[1, 0].plot(
        macro_df.index, macro_df['Unemployment Rate'] * 100, color='purple', alpha=0.3
    )
    axes[1, 0].plot(
        macro_df.index, macro_df['Unemployment Rate Smooth'] * 100, color='purple', linewidth=2, label='Trend'
    )
    axes[1, 0].set_title('Unemployment Rate (%)')
    axes[1, 0].set_xlabel('Tick')
    axes[1, 0].set_ylabel('% of Households Unemployed')
    axes[1, 0].set_ylim(0, 105)
    axes[1, 0].legend()
    axes[1, 0].grid(True)

    # Plot 4: Absolute Employment
    axes[1, 1].plot(
        macro_df.index, macro_df['Total Employed'], color='orange', alpha=0.3
    )
    axes[1, 1].plot(
        macro_df.index, macro_df['Total Employed Smooth'], color='orange', linewidth=2, label='Trend'
    )
    axes[1, 1].set_title('Total Number of Employed Households')
    axes[1, 1].set_xlabel('Tick')
    axes[1, 1].set_ylabel('Workers')
    axes[1, 1].legend()
    axes[1, 1].grid(True)

    plt.tight_layout(rect=[0, 0.03, 1, 0.95])  # Adjust layout for suptitle

    # Save the plot
    output_path: str = os.path.join(DATA_DIR, "macro_visualization.png")
    plt.savefig(output_path)
    print(f"Visualization saved to {output_path}")

    # Show the plot if running interactively
    try:
        plt.show()
    except Exception as e:
        print(
            "Could not display plot interactively "
            f"(this is normal in headless environments): {e}"
        )


if __name__ == "__main__":
    if not os.path.exists(CSV_FILE):
        print(f"Error: Could not find {CSV_FILE}. Run the Go simulation first.")
        exit(1)

    print("Loading simulation data...")
    # Read CSV. Employer might have NaNs depending on how it's written.
    df_raw: pd.DataFrame = pd.read_csv(CSV_FILE)
    df_raw['Employer'] = df_raw['Employer'].fillna(0).astype(int)
    if 'Price' in df_raw.columns:
        df_raw['Price'] = df_raw['Price'].fillna(0).astype(float)

    print("Calculating macroeconomic indicators...")
    macro_data_df: pd.DataFrame = calculate_macro_variables(df_raw)

    print("Generating plots...")
    plot_macro_variables(macro_data_df)
