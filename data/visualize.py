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

    # 1. Total Money Supply (M0 proxy) - Sum of all balances
    macro_data['Total Money Supply'] = grouped['Balance'].sum()

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

    # 3. Employment Rate
    # A household is employed if its Employer ID is not its own ID
    # Note: In the Go code, a household's employer is set to its own ID if
    # unemployed.
    def calc_employment(tick_df: pd.DataFrame) -> float:
        households = tick_df[tick_df['Type'] == 'Household']
        if households.empty:
            return 0.0
        employed = households[households['Employer'] != households['Id']]
        return float(len(employed) / len(households))

    macro_data['Employment Rate'] = grouped.apply(calc_employment)

    # 4. Total Employment (Absolute numbers)
    macro_data['Total Employed'] = grouped.apply(
        lambda d: len(
            d[(d['Type'] == 'Household') & (d['Employer'] != d['Id'])]
        )
    )

    return macro_data


def plot_macro_variables(macro_df: pd.DataFrame) -> None:
    """
    Generates subplots for the calculated macroeconomic variables.
    """
    # Set up the figure and axes
    fig, axes = plt.subplots(nrows=2, ncols=2, figsize=(14, 10))
    fig.suptitle('Monetary Simulation: Macroeconomic Indicators', fontsize=16)

    # Plot 1: Total Money Supply
    axes[0, 0].plot(
        macro_df.index, macro_df['Total Money Supply'], color='black'
    )
    axes[0, 0].set_title('Total Money Supply in Economy')
    axes[0, 0].set_xlabel('Tick')
    axes[0, 0].set_ylabel('Total Balance')
    axes[0, 0].grid(True)

    # Plot 2: Wealth Distribution
    axes[0, 1].plot(
        macro_df.index, macro_df['Household Wealth'],
        label='Households', color='blue'
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Firm Wealth'],
        label='Firms', color='green'
    )
    axes[0, 1].plot(
        macro_df.index, macro_df['Bank Wealth'],
        label='Bank', color='red'
    )
    axes[0, 1].set_title('Wealth Distribution by Sector')
    axes[0, 1].set_xlabel('Tick')
    axes[0, 1].set_ylabel('Total Balance')
    axes[0, 1].legend()
    axes[0, 1].grid(True)

    # Plot 3: Employment Rate
    axes[1, 0].plot(
        macro_df.index, macro_df['Employment Rate'] * 100, color='purple'
    )
    axes[1, 0].set_title('Employment Rate (%)')
    axes[1, 0].set_xlabel('Tick')
    axes[1, 0].set_ylabel('% of Households Employed')
    axes[1, 0].set_ylim(0, 105)
    axes[1, 0].grid(True)

    # Plot 4: Absolute Employment
    axes[1, 1].plot(
        macro_df.index, macro_df['Total Employed'], color='orange'
    )
    axes[1, 1].set_title('Total Number of Employed Households')
    axes[1, 1].set_xlabel('Tick')
    axes[1, 1].set_ylabel('Workers')
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

    print("Calculating macroeconomic indicators...")
    macro_data_df: pd.DataFrame = calculate_macro_variables(df_raw)

    print("Generating plots...")
    plot_macro_variables(macro_data_df)
